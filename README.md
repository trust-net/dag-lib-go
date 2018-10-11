# dag-lib-go
Go library for DAG protocol

## How to use DLT stack library
### Create configuration
create a `p2p.Config` instance with following values:
```
type Config struct {
	// path to private key for p2p layer node, if file does not exists then a new
	// key is created and persisted to that file, otherwise key is read from that file
	KeyFile string `json:"key_file"       gencodec:"required"`

	// type of private key for p2p layer node
	// (currently only "ECDSA_S256" is supported)
	KeyType string  `json:"key_type"       gencodec:"required"`

	// maximum number of peers that can be connected
	// (It must be greater than zero)
	MaxPeers int `json:"max_peers"       gencodec:"required"`

	// the node name of this server
	Name string  `json:"node_name"       gencodec:"required"`

	// A list of DEVp2p node addresses to establish connectivity
	// with the rest of the network.
	Bootnodes []string `json:"boot_nodes"`

	// If the port is zero, then operating system will pick a port
	Port string `json:"listen_port"`

	// If set to true, the listening port is made available to the
	// Internet.
	NAT bool
}
```
These values can be read from a file, using the field names as specified in the `json` directive against each of them. An sample file is:
```
{
	"key_file": "name of file to persist private key for the node",
	"key_type": "ECDSA_S256",
	"max_peers": 2,
	"proto_name": "test",
	"node_name": "test node 1",
	"listen_port": "50883",
	"boot_nodes": ["enode://c3da24ed70538b731b9734e4e0b8206e441089ab4fcd1d0faadb1031e736491b70de0b70e1d581958b28eb43444491b3b9091bd8a81d1767bf7d4ebc3e7bd108@127.0.0.1:50438"]
}
```

### Instantiate DLT stack
Use `stack.NewDltStack(conf p2p.Config, db db.Database)` method to instantiate a DLT stack controller. This takes two arguments:
* a `p2p.Config` structure with parameters as described above
* an implementation of `db.Database` implementation, that will be used by DLT stack to save/retrieve/persist data

### Register application with DLT stack
If running an application on the DLT stack, then register the application with the DLT stack using the `stack.DLT.Register(app stack.AppConfig, peerHandler stack.PeerApprover, txHandler stack.NetworkTxApprover) error` method. This takes following arguments:
* a `stack.AppConfig` structure with application details (name of aplication, shard ID of the application, version of the application)
* a `stack.PeerApprover` type function that will be called for confirming if application wants to receive transaction from a peer. This method will be called for each transaction (since we don't know when a peer may get blacklisted)
* a `stack.NetworkTxApprover` type function that will be called to accept/process a transaction from a peer. If this method returns back a non-nil error, then transaction will not be forwarded to other network peers

> This step is optional because a deployment may choose to run in "headless" mode, in which case it will not process any application transactions and will only participate in the transaction endorsement process, to provide network security.

### Start the DLT stack
Use `stack.DLT.Start()` method for DLT stack to start discovering other nodes on the network and start listening to messages from connected peers. 

### Process transactions from network peers
If application had registered with DLT stack with appropriate callback methods, then after DLT stack is started, whenever a new network transaction is received, it is processed as following:
1. first the application provided `stack.PeerApprover` is called with transaction submitter node's ID, to check if application is willing to accept transactions from the remote application instance running on that node
2. second (if peer is approved) the application provided `stack.NetworkTxApprover` is called with transaction details, Application is suppose to return back an error if transaction was not accepted

### Stop DLT Stack
Once application execution completes (either due to application shutdown, or any other reason), call the `stack.DLT.Stop()` method to disconnect from all connected network peers.

## Example
An example test program that implement a distributed network counter using DLT stack library is provided under the `tests/countr/app.go`. It can be used as following:
### Build test application
```
(cd tests/countr/; go build)
```
### Create config file
Use the example mentioned above to create config files for each instance that need to be run. Please make sure:
* port is unique in each copy
* key file name is unique in each copy

### Run the test application
Start each instance of the test application with its corresponding config file:
```
./tests/countr/countr -config config1.json
```

> As per iteration 1 (Issue #22), all instances of the nodes need to be started before transactions submission. Nodes don't yet have syncing capability. Also, nodes do not yet persist state across reboot.

### Send counter transaction
Use the application's CLI to submit transactions as following:
```
Command: incr cntr1 
adding transaction: incr cntr1 1

Command: decr cntr2
adding transaction: decr cntr2 1

Command: 
```
Use application's CLI to query a counter's value:
```
Command: countr cntr1
     cntr1: 1
Command: countr cntr2
     cntr2: -1
Command: countr cntr3
     cntr3: not found
Command: 
```

### Shutdown application
```
Command: quit
Shutdown cleanly
```


 