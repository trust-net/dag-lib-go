// Copyright 2019 The trust-net Authors
// A driver application to test DLT Stack library's double spending resolution
package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trust-net/dag-lib-go/common"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/state"
	"math/big"
	"os"
	"strconv"
	"strings"
)

var (
	AppName   = "test-driver-for-double-spending"
	AppShard  = []byte(AppName)
	cmdPrompt = "SPENDR: "
)

var key *ecdsa.PrivateKey

var submitter []byte

var lastTx = [64]byte{}
var lastSeq = uint64(0)
var dltStack stack.DLT

// Transaction Ops
type Ops struct {
	// op code
	Code uint64
	// serialized arguments
	Args []byte
}

// Op Codes for supported transactions
const (
	OpCodeCreate uint64 = iota + 0x01
	OpCodeXferValue
)

// arguments for OpCodeCreate
type ArgsCreate struct {
	// resource name
	Name string
	// initial value
	Value int64
}

// arguments for OpCodeXferValue
type ArgsXferValue struct {
	// xfer source name
	Source string
	// xfer destination name
	Destination string
	// xfer value
	Value int64
}

func sign(tx dto.Transaction, txPayload []byte) dto.Transaction {
	// sign the test payload using SHA512 hash and ECDSA private key
	type signature struct {
		R *big.Int
		S *big.Int
	}
	s := signature{}
	hash := sha512.Sum512(txPayload)
	s.R, s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	tx.Self().Payload = txPayload
	tx.Self().Signature, _ = common.Serialize(s)
	return tx
}

func makeTransaction(a *dto.Anchor, opCode uint64, arg interface{}) {
	if a == nil {
		fmt.Printf("Error submitting transaction: no anchor!!!\n")
		return
	}
	op := Ops{
		Code: opCode,
	}
	op.Args, _ = common.Serialize(arg)
	txPayload, _ := common.Serialize(op)
	tx := sign(dto.NewTransaction(a), txPayload)
	if err := dltStack.Submit(tx); err != nil {
		fmt.Printf("Error submitting transaction: %s\n", err)
	} else {
		lastTx = tx.Id()
		lastSeq += 1
	}
}

func submitTransaction(a *dto.Anchor, opCode uint64, arg interface{}) dto.Transaction {
	if a == nil {
		return nil
	}
	op := Ops{
		Code: opCode,
	}
	op.Args, _ = common.Serialize(arg)
	txPayload, _ := common.Serialize(op)
	return sign(dto.NewTransaction(a), txPayload)
}

func scanCreateArgs(scanner *bufio.Scanner) (args []ArgsCreate) {
	nextToken := func() (*string, int, bool) {
		if !scanner.Scan() {
			return nil, 0, false
		}
		word := scanner.Text()
		if value, err := strconv.Atoi(word); err == nil {
			return nil, value, true
		} else {
			return &word, 0, true
		}
	}
	args = make([]ArgsCreate, 0)
	currArg := ArgsCreate{}
	readName := false
	for {
		name, value, success := nextToken()

		if !success {
			if readName {
				args = append(args, currArg)
			}
			return
		} else if name == nil && currArg.Name == "" {
			return
		}

		if name != nil {
			if readName {
				args = append(args, currArg)
			}
			currArg = ArgsCreate{}
			currArg.Name = *name
			currArg.Value = 0
			readName = true
		} else {
			currArg.Value = int64(value)
			args = append(args, currArg)
			currArg = ArgsCreate{}
			readName = false
		}
	}
}

func handleOpCodeCreate(tx dto.Transaction, ws state.State, op Ops) error {
	// parse the args
	arg := ArgsCreate{}
	if err := common.Deserialize(op.Args, &arg); err != nil {
		return err
	}
	fmt.Printf("Transaction to create a resource: %s = %d\n", arg.Name, arg.Value)
	// validate: resource should not already exist
	if r, err := ws.Get([]byte(arg.Name)); err == nil {
		fmt.Printf("ERROR: attempt to create an existing resource: %s\nOwner: %x", arg.Name, r.Owner)
		return fmt.Errorf("Resource already exists")
	}
	// create the resource
	r := state.Resource{
		Key:   []byte(arg.Name),
		Owner: tx.Anchor().Submitter,
		Value: common.Uint64ToBytes(uint64(arg.Value)),
	}
	// create resource in world state
	return ws.Put(&r)
}

func handleOpCodeXferValue(tx dto.Transaction, ws state.State, op Ops) error {
	// parse the args
	arg := ArgsXferValue{}
	if err := common.Deserialize(op.Args, &arg); err != nil {
		return err
	}
	fmt.Printf("Transaction to xfer '%s' ---%d--> '%s'\n", arg.Source, arg.Value, arg.Destination)
	fmt.Printf("Shard Seq: '%x', Weight: '%x', Parent: %x\n", tx.Anchor().ShardSeq, tx.Anchor().Weight, tx.Anchor().ShardParent)
	fmt.Printf("Submt Seq: '%x', Parent: %x\n", tx.Anchor().SubmitterSeq, tx.Anchor().SubmitterLastTx)
	// validate: resources should already exist
	var from, to *state.Resource
	var err error
	if from, err = ws.Get([]byte(arg.Source)); err != nil {
		fmt.Printf("ERROR: attempt to xfer value from a non existing resource: %s\nSubmitter: %x\n", arg.Source, tx.Anchor().Submitter)
		return fmt.Errorf("Resource does not exists")
	}
	if to, err = ws.Get([]byte(arg.Destination)); err != nil {
		fmt.Printf("ERROR: attempt to xfer value to a non existing resource: %s\nSubmitter: %x\n", arg.Destination, tx.Anchor().Submitter)
		return fmt.Errorf("Resource does not exists")
	}
	// validate: source resource must be owned by submitter
	if string(tx.Anchor().Submitter) != string(from.Owner) {
		fmt.Printf("ERROR: attempt to xfer value from unauthorized resource: %s\nOwner: %x\nSubmitter: %x\n", arg.Source, from.Owner, tx.Anchor().Submitter)
		return fmt.Errorf("Resource not owned")
	}
	// validate: xfer value should not be more than source resource's value
	fromValue := int64(common.BytesToUint64(from.Value))
	if fromValue < arg.Value {
		fmt.Printf("ERROR: attempt to xfer excess value: %d\nResource value: %d\nSubmitter: %x\n", arg.Value, fromValue, tx.Anchor().Submitter)
		return fmt.Errorf("Resource insufficient")
	}
	// validate: xfer value cannot be less than 1 (i.e. cannot make negative transaction from other people's resource)
	if arg.Value < 1 {
		fmt.Printf("ERROR: attempt to make deduction from other people: %d\nSubmitter: %x\n", arg.Value, tx.Anchor().Submitter)
		return fmt.Errorf("Negative transaction")
	}
	// make the transaction
	toValue := int64(common.BytesToUint64(to.Value))
	fromValue, toValue = fromValue-arg.Value, toValue+arg.Value
	from.Value = common.Uint64ToBytes(uint64(fromValue))
	to.Value = common.Uint64ToBytes(uint64(toValue))
	// update resources in world state
	if err := ws.Put(from); err != nil {
		fmt.Printf("Error in updating '%s' with world state: %s\n", from.Key, err)
		return err
	}
	if err := ws.Put(to); err != nil {
		fmt.Printf("Error in updating '%s' with world state: %s\n", to.Key, err)
		return err
	}
	return nil
}

func txHandler(tx dto.Transaction, state state.State) error {
	fmt.Printf("\n")
	defer fmt.Printf("\n%s", cmdPrompt)
	op := Ops{}
	if err := common.Deserialize(tx.Self().Payload, &op); err != nil {
		fmt.Printf("Invalid TX from %x\n%s", tx.Anchor().NodeId, err)
		return err
	}
	switch op.Code {
	case OpCodeCreate:
		return handleOpCodeCreate(tx, state, op)
	case OpCodeXferValue:
		return handleOpCodeXferValue(tx, state, op)
	default:
		fmt.Printf("Unknown Op Code: %d\n", op.Code)
		return fmt.Errorf("Unknown Op Code: %d", op.Code)
	}
}

// main CLI loop
func cli(dlt stack.DLT) error {
	dltStack = dlt
	if err := dlt.Start(); err != nil {
		return err
	} else if err := dlt.Register(AppShard, AppName, txHandler); err != nil {
		return err
	}
	for {
		fmt.Printf(cmdPrompt)
		lineScanner := bufio.NewScanner(os.Stdin)
		for lineScanner.Scan() {
			line := lineScanner.Text()
			if len(line) != 0 {
				wordScanner := bufio.NewScanner(strings.NewReader(line))
				wordScanner.Split(bufio.ScanWords)
				for wordScanner.Scan() {
					cmd := wordScanner.Text()
					switch cmd {
					case "quit":
						fallthrough
					case "q":
						dlt.Stop()
						return nil
					case "value":
						fallthrough
					case "show":
						fallthrough
					case "v":
						hasNext := wordScanner.Scan()
						oneDone := false
						for hasNext {
							name := wordScanner.Text()
							if len(name) != 0 {
								if oneDone {
									fmt.Printf("\n")
								} else {
									oneDone = true
								}
								// get current network counter value from world state
								if r, err := dlt.GetState([]byte(name)); err == nil {
									value := int64(common.BytesToUint64(r.Value))
									common.Deserialize(r.Value, &value)
									fmt.Printf("% 10s: %d", name, value)
								} else {
									fmt.Printf("% 10s: %s", name, err)
								}
							}
							hasNext = wordScanner.Scan()
						}
						if !oneDone {
							fmt.Printf("usage: value <name> ...\n")
						}
					case "create":
						fallthrough
					case "c":
						args := scanCreateArgs(wordScanner)
						if len(args) == 0 {
							fmt.Printf("usage: create <resource name> [<initial value>] ...\n")
						} else {
							for _, arg := range args {
								fmt.Printf("adding transaction: create %s %d\n", arg.Name, arg.Value)
								makeTransaction(dlt.Anchor(submitter, lastSeq+1, lastTx), OpCodeCreate, arg)
							}
						}
					case "info":
						for wordScanner.Scan() {
							continue
						}
						if a := dlt.Anchor(submitter, lastSeq+1, lastTx); a == nil {
							fmt.Printf("failed to get any info...\n")
						} else {
							fmt.Printf("ShardId: %s\n", a.ShardId)
							fmt.Printf("Next Seq: %d\n", a.ShardSeq)
							fmt.Printf("Parent: %x\n", a.ShardParent)
							fmt.Printf("NodeId: %x\n", a.NodeId)
						}
					case "xfer":
						arg := ArgsXferValue{}
						if wordScanner.Scan() {
							arg.Source = wordScanner.Text()
						}
						if wordScanner.Scan() {
							value, _ := strconv.Atoi(wordScanner.Text())
							arg.Value = int64(value)
						}
						if wordScanner.Scan() {
							arg.Destination = wordScanner.Text()
						}
						if len(arg.Source) != 0 && len(arg.Destination) != 0 && arg.Value > 0 {
							fmt.Printf("adding transaction: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(dlt.Anchor(submitter, lastSeq+1, lastTx), OpCodeXferValue, arg)
						} else {
							fmt.Printf("usage: xfer <owned resource name> <xfer value> <recipient resource name>\n")
						}
					case "dupe":
						arg := ArgsXferValue{}
						if wordScanner.Scan() {
							arg.Source = wordScanner.Text()
						}
						if wordScanner.Scan() {
							value, _ := strconv.Atoi(wordScanner.Text())
							arg.Value = int64(value)
						}
						var dest1, dest2 string
						if wordScanner.Scan() {
							dest1 = wordScanner.Text()
						}
						if wordScanner.Scan() {
							dest2 = wordScanner.Text()
						}
						if len(arg.Source) != 0 && len(dest1) != 0 && len(dest2) != 0 && arg.Value > 0 {
							arg.Destination = dest1
							anchor1 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							anchor2 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding transaction #1: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(anchor1, OpCodeXferValue, arg)
							arg.Destination = dest2
							fmt.Printf("adding transaction #2: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(anchor2, OpCodeXferValue, arg)
						} else {
							fmt.Printf("usage: dupe <owned resource name> <xfer value> <recipient 1> <recipient 2>\n")
						}
					case "double":
						arg := ArgsXferValue{}
						if wordScanner.Scan() {
							arg.Source = wordScanner.Text()
						}
						if wordScanner.Scan() {
							value, _ := strconv.Atoi(wordScanner.Text())
							arg.Value = int64(value)
						}
						var dest1, dest2 string
						if wordScanner.Scan() {
							dest1 = wordScanner.Text()
						}
						if wordScanner.Scan() {
							dest2 = wordScanner.Text()
						}
						if len(arg.Source) != 0 && len(dest1) != 0 && len(dest2) != 0 && arg.Value > 0 {
							arg.Destination = dest1
							anchor1 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding transaction #1: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(anchor1, OpCodeXferValue, arg)
							arg.Destination = dest2
							anchor2 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding transaction #2: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(anchor2, OpCodeXferValue, arg)
						} else {
							fmt.Printf("usage: double <owned resource name> <xfer value> <recipient 1> <recipient 2>\n")
						}
					default:
						fmt.Printf("Unknown Command: %s", cmd)
						for wordScanner.Scan() {
							fmt.Printf(" %s", wordScanner.Text())
						}
						break
					}
				}
			}
			fmt.Printf("\n%s", cmdPrompt)
		}
	}
	dltStack = nil
	return nil
}

func main() {
	fileName := flag.String("config", "", "config file name")
	flag.Parse()
	if len(*fileName) == 0 {
		fmt.Printf("Missing required parameter \"config\"\n")
		return
	}
	// open the config file
	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Printf("Failed to open config file: %s\n", err)
		return
	}
	data := make([]byte, 2048)
	// read config data from file
	config := p2p.Config{}
	if count, err := file.Read(data); err == nil {
		data = data[:count]
		// parse json data into structure
		if err := json.Unmarshal(data, &config); err != nil {
			fmt.Printf("Failed to parse config data: %s\n", err)
			return
		}
	} else {
		fmt.Printf("Failed to read config file: %s\n", err)
		return
	}

	// create a new ECDSA key for submitter client
	key, _ = crypto.GenerateKey()
	submitter = crypto.FromECDSAPub(&key.PublicKey)

	// instantiate the DLT stack
	if dlt, err := stack.NewDltStack(config, db.NewInMemDbProvider()); err != nil {
		fmt.Printf("Failed to create DLT stack: %s", err)
	} else if err = cli(dlt); err != nil {
		fmt.Printf("Error in CLI: %s", err)
	} else {
		fmt.Printf("Shutdown cleanly")
	}
	fmt.Printf("\n")
}
