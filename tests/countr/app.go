// Copyright 2018 The trust-net Authors
// A network counter application to test DLT Stack library
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/state"
	"github.com/trust-net/dag-lib-go/common"
	"os"
	"strconv"
	"strings"
)

var commands = map[string][2]string{
	"countr": {"usage: countr <countr name> ...", "view a counter value"},
	"incr":   {"usage: incr <countr name> [<integer>] ...", "increment one or more counters"},
	"decr":   {"usage: decr <countr name> [<integer>] ...", "decrement one or more counters"},
	"info":   {"usage: info", "get current shard tip"},
	"join":   {"usage: join <shard id> [<name>]", "join a shard (a unique string)"},
	"leave":  {"usage: leave", "leave from a registered shard (run as headless, default behavior)"},
}

var cmdPrompt = "<headless>: "

var shardId []byte

var submitter *dto.Submitter

type testTx struct {
	Op     string
	Target string
	Delta  int64
}

func incrementTx(name string, delta int) *dto.TxRequest {
	op := testTx{
		Op:     "incr",
		Target: name,
		Delta:  int64(delta),
	}
	txPayload, _ := common.Serialize(op)
	return submitter.NewRequest(string(txPayload))
}

func decrementTx(name string, delta int) *dto.TxRequest {
	op := testTx{
		Op:     "decr",
		Target: name,
		Delta:  int64(delta),
	}
	txPayload, _ := common.Serialize(op)
	return submitter.NewRequest(string(txPayload))
}

type op struct {
	name  string
	delta int
}

func scanOps(scanner *bufio.Scanner) (ops []op) {
	nextToken := func() (*string, int, bool) {
		if !scanner.Scan() {
			return nil, 0, false
		}
		word := scanner.Text()
		if delta, err := strconv.Atoi(word); err == nil {
			return nil, delta, true
		} else {
			return &word, 0, true
		}
	}
	ops = make([]op, 0)
	currOp := op{}
	readName := false
	for {
		name, delta, success := nextToken()

		if !success {
			if readName {
				ops = append(ops, currOp)
			}
			return
		} else if name == nil && currOp.name == "" {
			return
		}

		if name != nil {
			if readName {
				ops = append(ops, currOp)
			}
			currOp = op{}
			currOp.name = *name
			currOp.delta = 1
			readName = true
		} else {
			currOp.delta = delta
			ops = append(ops, currOp)
			currOp = op{}
			readName = false
		}
	}
}

func applyDelta(name string, delta int, s state.State) int64 {
	last := int64(0)
	// fetch resource from world state
	r := &state.Resource{
		Key: []byte(name),
	}
	if read, err := s.Get(r.Key); err == nil {
		common.Deserialize(read.Value, &last)
	}
	last += int64(delta)
	r.Value, _ = common.Serialize(last)
	s.Put(r)
	return last
}

func txHandler(tx dto.Transaction, state state.State) error {
	fmt.Printf("\n")
	op := testTx{}
	if err := common.Deserialize(tx.Request().Payload, &op); err != nil {
		fmt.Printf("Invalid TX from %x\n%s", tx.Anchor().NodeId, cmdPrompt)
		return err
	}
	fmt.Printf("TX: %s %s %d\n", op.Op, op.Target, op.Delta)
	delta := 0
	switch op.Op {
	case "incr":
		delta = int(op.Delta)
	case "decr":
		delta = int(-op.Delta)
	}
	fmt.Printf("%s --> %d\n%s", op.Target, applyDelta(op.Target, delta, state), cmdPrompt)
	return nil
}

// main CLI loop
func cli(dlt stack.DLT) error {
	if err := dlt.Start(); err != nil {
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
					case "countr":
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
									var last int64
									common.Deserialize(r.Value, &last)
									fmt.Printf("% 10s: %d", name, last)
								} else {
									fmt.Printf("% 10s: %s", name, err)
								}
							}
							hasNext = wordScanner.Scan()
						}
						if !oneDone {
							fmt.Printf("%s\n", commands["countr"][1])
							fmt.Printf("%s\n", commands["countr"][0])
						}
					case "incr":
						ops := scanOps(wordScanner)
						if len(ops) == 0 {
							fmt.Printf("%s\n", commands[cmd][1])
							fmt.Printf("%s\n", commands[cmd][0])
						} else {
							for _, op := range ops {
								fmt.Printf("adding transaction: incr %s %d\n", op.name, op.delta)
								if tx, err := dlt.Submit(incrementTx(op.name, op.delta)); err != nil {
									fmt.Printf("Error submitting transaction: %s\n", err)
								} else {
									submitter.Seq += 1
									submitter.LastTx = tx.Id()
								}
							}
						}
					case "decr":
						ops := scanOps(wordScanner)
						if len(ops) == 0 {
							fmt.Printf("%s\n", commands[cmd][1])
							fmt.Printf("%s\n", commands[cmd][0])
						} else {
							for _, op := range ops {
								fmt.Printf("adding transaction: decr %s %d\n", op.name, op.delta)
								if tx, err := dlt.Submit(decrementTx(op.name, op.delta)); err != nil {
									fmt.Printf("Error submitting transaction: %s\n", err)
								} else {
									submitter.Seq += 1
									submitter.LastTx = tx.Id()
								}
							}
						}
					case "info":
						for wordScanner.Scan() {
							continue
						}
						if a := dlt.Anchor(submitter.Id, submitter.Seq, submitter.LastTx); a == nil {
							fmt.Printf("failed to get any info...\n")
						} else {
							fmt.Printf("Next Seq: %d\n", a.ShardSeq)
							fmt.Printf("Parent: %x\n", a.ShardParent)
							fmt.Printf("NodeId: %x\n", a.NodeId)
						}
					case "join":
						if !wordScanner.Scan() {
							fmt.Printf("%s\n", commands[cmd][1])
							fmt.Printf("%s\n", commands[cmd][0])
							break
						}
						name := wordScanner.Text()
						shardId = []byte(name)
						if wordScanner.Scan() {
							name = wordScanner.Text()
						}
						if err := dlt.Register([]byte(shardId), name, txHandler); err != nil {
							fmt.Printf("Error registering app: %s\n", err)
						} else {
							cmdPrompt = "<" + name + ">: "
							submitter.ShardId = shardId
						}
					case "leave":
						for wordScanner.Scan() {
							continue
						}
						if err := dlt.Unregister(); err != nil {
							fmt.Printf("Error during un-registering app: %s\n", err)
						}
						cmdPrompt = "<headless>: "
						submitter.ShardId = nil
					default:
						fmt.Printf("Unknown Command: %s", cmd)
						for wordScanner.Scan() {
							fmt.Printf(" %s", wordScanner.Text())
						}
						fmt.Printf("\n\nAccepted commands...\n")
						isFirst := true
						for k, _ := range commands {
							if !isFirst {
								fmt.Printf(", ")
							} else {
								isFirst = false
							}
							fmt.Printf("\"%s\"", k)
						}
						fmt.Printf("\n")
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

	// create a new submitter client
	submitter = dto.TestSubmitter()

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
