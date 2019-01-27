// Copyright 2019 The trust-net Authors
// A driver application to test DLT Stack library's double spending resolution
package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

var commands = map[string][2]string{
	"show":   {"usage: show <countr name> ...", "show one or more counter's value"},
	"create": {"usage: create <resource name> [<initial value>] ...", "create one or more counters with optional initial credits"},
	"xfer":   {"usage: xfer <owned counter name> <xfer value> <recipient counter name>...", "transfer credits from one counter to another"},
	"info":   {"usage: info", "get current shard tips from local and remote nodes"},
	"xover":  {"usage: xover <owned resource name> <xfer value> <recipient resource name>", "submit a transaction that has anchor from one node, but is submitted to another node"},
	"quit":   {"usage: quit", "leave application and shutdown"},
	"dupe":   {"usage: dupe <owned resource name> <xfer value> <recipient 1> <recipient 2>", "submit two double spending transactions using same tip"},
	"double": {"usage: double <owned counter name> <xfer value> <recipient 1 counter> <recipient 2 countr>", "submit two double spending transactions on local node"},
	"multi":  {"usage: multi <owned resource name> <xfer value> <recipient resource name>", "submit a redundant transactions on two different nodes"},
	"split":  {"usage: split <owned resource name> <xfer value> <recipient 1> <recipient 2>", "submit two double spending transactions on two different nodes"},
}

var (
	AppName   = "test-driver-for-double-spending"
	AppShard  = []byte(AppName)
	cmdPrompt = "SPENDR: "
)

var key *ecdsa.PrivateKey

var submitter []byte

var lastTx = [64]byte{}
var lastSeq = uint64(0)

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
	hash := sha256.Sum256(append(common.Uint64ToBytes(tx.Anchor().SubmitterSeq), txPayload...))
	s.R, s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	tx.Self().Payload = txPayload
//	tx.Self().Signature, _ = common.Serialize(s)
	tx.Self().Signature = append(s.R.Bytes(), s.S.Bytes()...)
	return tx
}

func makeTransaction(dltStack stack.DLT, a *dto.Anchor, txPayload []byte) {
	if a == nil {
		fmt.Printf("Error submitting transaction: no anchor!!!\n")
		return
	}
	tx := sign(dto.NewTransaction(a), txPayload)
	if err := dltStack.Submit(tx); err != nil {
		fmt.Printf("Error submitting transaction: %s\n", err)
	} else {
		lastTx = tx.Id()
		lastSeq += 1
	}
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
	//	fmt.Printf("Transaction to create a resource: %s = %d\n", arg.Name, arg.Value)
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
	//	fmt.Printf("Transaction to xfer '%s' ---%d--> '%s'\n", arg.Source, arg.Value, arg.Destination)
	//	fmt.Printf("Shard Seq: '%x', Weight: '%x', Parent: %x\n", tx.Anchor().ShardSeq, tx.Anchor().Weight, tx.Anchor().ShardParent)
	//	fmt.Printf("Submt Seq: '%x', Parent: %x\n", tx.Anchor().SubmitterSeq, tx.Anchor().SubmitterLastTx)
	// validate: resources should already exist
	var from, to *state.Resource
	var err error
	// first deduct from source and update world state
	if from, err = ws.Get([]byte(arg.Source)); err != nil {
		fmt.Printf("ERROR: attempt to xfer value from a non existing resource: %s\nSubmitter: %x\n", arg.Source, tx.Anchor().Submitter)
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
	// deduct from source
	from.Value = common.Uint64ToBytes(uint64(fromValue - arg.Value))
	// update world state
	if err := ws.Put(from); err != nil {
		fmt.Printf("Error in updating '%s' with world state: %s\n", from.Key, err)
		return err
	}
	// now fetch destination
	if to, err = ws.Get([]byte(arg.Destination)); err != nil {
		fmt.Printf("ERROR: attempt to xfer value to a non existing resource: %s\nSubmitter: %x\n", arg.Destination, tx.Anchor().Submitter)
		return fmt.Errorf("Resource does not exists")
	}
	// add value to destination resource
	toValue := int64(common.BytesToUint64(to.Value))
	to.Value = common.Uint64ToBytes(uint64(toValue + arg.Value))
	// update world state
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

var dlt, remoteDlt, localDlt stack.DLT

func doGetResource(key string) ([]byte, uint64, error) {
	// get current network counter value from world state
	if r, err := dlt.GetState([]byte(key)); err == nil {
		value := common.BytesToUint64(r.Value)
		return r.Owner, value, nil
	} else {
		return nil, 0, err
	}
}

func doRequestAnchor(id []byte, seq uint64, lastTx [64]byte) *dto.Anchor {
	return dlt.Anchor(id, seq, lastTx)
}

func doSubmitTransaction(tx dto.Transaction) error {
	return dlt.Submit(tx)
}

func makeXferValuePayload(source, destination string, value int64) []byte {
	op := Ops{
		Code: OpCodeXferValue,
	}
	args := ArgsXferValue{
		Source:      source,
		Destination: destination,
		Value:       value,
	}
	op.Args, _ = common.Serialize(args)
	txPayload, _ := common.Serialize(op)
	return txPayload
}

func makeResourceCreationPayload(key string, value int64) []byte {
	op := Ops{
		Code: OpCodeCreate,
	}
	args := ArgsCreate{
		Name:  key,
		Value: value,
	}
	op.Args, _ = common.Serialize(args)
	txPayload, _ := common.Serialize(op)
	return txPayload
}

// main CLI loop
func cli(local, remote stack.DLT) error {
	dlt, remoteDlt, localDlt = local, remote, local

	if err := localDlt.Start(); err != nil {
		return err
	} else if err := localDlt.Register(AppShard, AppName, txHandler); err != nil {
		return err
	} else if err := remoteDlt.Start(); err != nil {
		return err
	} else if err := remoteDlt.Register(AppShard, AppName, txHandler); err != nil {
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
							key := wordScanner.Text()
							if len(key) != 0 {
								if oneDone {
									fmt.Printf("\n")
								} else {
									oneDone = true
								}
								// get current network counter value from world state
								if owner, value, err := doGetResource(key); err == nil {
									fmt.Printf("%x [% 10s]: %d", owner, key, value)
								} else {
									fmt.Printf("% 10s: %s", key, err)
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
								makeTransaction(dlt, dlt.Anchor(submitter, lastSeq+1, lastTx), makeResourceCreationPayload(arg.Name, arg.Value))
							}
						}
					case "info":
						for wordScanner.Scan() {
							continue
						}
						if a := localDlt.Anchor([]byte("dummy"), 0x01, [64]byte{}); a == nil {
							fmt.Printf("failed to get any info from local node...\n")
						} else {
							fmt.Printf("LOCAL ShardId: %s\n", a.ShardId)
							fmt.Printf("Submitter Id : %x\n", submitter)
							fmt.Printf("LOCAL Next Seq: %d\n", a.ShardSeq)
							fmt.Printf("LOCAL Weight: %d\n", a.Weight)
							fmt.Printf("LOCAL Parent: %x\n", a.ShardParent)
						}
						if a := remoteDlt.Anchor([]byte("dummy"), 0x01, [64]byte{}); a == nil {
							fmt.Printf("failed to get any info from remote node...\n")
						} else {
							fmt.Printf("REMOT Parent: %x\n", a.ShardParent)
							fmt.Printf("REMOT Next Seq: %d\n", a.ShardSeq)
							fmt.Printf("REMOT Weight: %d\n", a.Weight)
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
							makeTransaction(dlt, dlt.Anchor(submitter, lastSeq+1, lastTx), makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
						} else {
							fmt.Printf("usage: xfer <owned resource name> <xfer value> <recipient resource name>\n")
						}
					case "sign":
						var payload string
						var nonce int
						if wordScanner.Scan() {
							nonce, _ = strconv.Atoi(wordScanner.Text())
						}
						if wordScanner.Scan() {
							payload = wordScanner.Text()
						}
						if nonce > 0 && len(payload) != 0 {
							if bytes, err := base64.StdEncoding.DecodeString(payload); err != nil {
								fmt.Printf("Invalid base64 payload: %s\n", err)
							} else {
								// sign payload using CLI's submitter
								tx := dto.TestTransaction()
								tx.Anchor().SubmitterSeq = uint64(nonce)
								sign(tx, bytes)
								// print the base64 encoded signature
								fmt.Printf("Signature: %s\n", base64.StdEncoding.EncodeToString(tx.Self().Signature))
							}
						} else {
							fmt.Printf("usage: sign <nonce> <base64 encoded payload>\n")
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
							makeTransaction(dlt, anchor1, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
							arg.Destination = dest2
							fmt.Printf("adding transaction #2: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(dlt, anchor2, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
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
							anchor2 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding transaction #1: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(dlt, anchor1, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
							arg.Destination = dest2
							//							anchor2 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding transaction #2: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(dlt, anchor2, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
						} else {
							fmt.Printf("usage: double <owned resource name> <xfer value> <recipient 1> <recipient 2>\n")
						}
					case "xover":
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
							makeTransaction(remoteDlt, localDlt.Anchor(submitter, lastSeq+1, lastTx), makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
						} else {
							fmt.Printf("usage: xover <owned resource name> <xfer value> <recipient resource name>\n")
						}
					case "multi":
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
							anchor1 := localDlt.Anchor(submitter, lastSeq+1, lastTx)
							anchor2 := remoteDlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding local transaction #1: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(localDlt, anchor1, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
							fmt.Printf("adding remote transaction #2: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(remoteDlt, anchor2, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
						} else {
							fmt.Printf("usage: multi <owned resource name> <xfer value> <recipient resource name>\n")
						}
					case "split":
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
							anchor1 := localDlt.Anchor(submitter, lastSeq+1, lastTx)
							anchor2 := remoteDlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding local transaction #1: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(localDlt, anchor1, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
							arg.Destination = dest2
							//							anchor2 := dlt.Anchor(submitter, lastSeq+1, lastTx)
							fmt.Printf("adding remote transaction #2: xfer %s %d %s\n", arg.Source, arg.Value, arg.Destination)
							makeTransaction(remoteDlt, anchor2, makeXferValuePayload(arg.Source, arg.Destination, arg.Value))
						} else {
							fmt.Printf("usage: split <owned resource name> <xfer value> <recipient 1> <recipient 2>\n")
						}
					default:
						fmt.Printf("Unknown Command: %s", cmd)
						for wordScanner.Scan() {
							fmt.Printf(" %s", wordScanner.Text())
						}
						fmt.Printf("\nUsages...\n")
						for k, v := range commands {
							fmt.Printf("\"%s\": %s\n%s\n", k, v[1], v[0])
						}
						break
					}
				}
			}
			fmt.Printf("\n%s", cmdPrompt)
		}
	}
	return nil
}

func main() {
	fileName := flag.String("config", "", "config file name")
	apiPort := flag.Int("apiPort", 0, "port for client API")
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

	// create a 2nd node config from original config
	config2 := p2p.Config{}
	config2 = config
	config2.KeyFile = "remoteKey.json"
	config2.Name = "remote-" + config.Name
	port, _ := strconv.Atoi(config.Port)
	config2.Port = strconv.Itoa(port + 100)

	// create a new ECDSA key for submitter client
	key, _ = crypto.GenerateKey()
	submitter = crypto.FromECDSAPub(&key.PublicKey)

	// start net server
	if err := StartServer(*apiPort); err != nil {
		fmt.Printf("Did not start client API: %s\n", err)
	}

	// instantiate two DLT stacks
	if localDlt, err := stack.NewDltStack(config, db.NewInMemDbProvider()); err != nil {
		fmt.Printf("Failed to create 1st DLT stack: %s", err)
	} else if remoteDlt, err := stack.NewDltStack(config2, db.NewInMemDbProvider()); err != nil {
		fmt.Printf("Failed to create 2nd DLT stack: %s", err)
	} else if err = cli(localDlt, remoteDlt); err != nil {
		fmt.Printf("Error in CLI: %s", err)
	} else {
		fmt.Printf("Shutdown cleanly")
	}
	fmt.Printf("\n")
}
