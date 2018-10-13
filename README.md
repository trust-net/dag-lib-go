# dag-lib-go
Go library for DAG protocol

## How to use DLT stack library
### Create configuration
Create a `p2p.Config` instance. These values can be read from a file, using the field names as specified in the `json` directive against each of them. A sample file is:
```
{
	"key_file": "name of file to persist private key for the node",
	"key_type": "ECDSA_S256",
	"max_peers": 2,
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

> As per iteration 2, all instances of the nodes need to be started before transactions submission. Nodes don't yet have syncing capability. Also, nodes do not yet persist state across reboot.

### Register application shard
Use the application's CLI to join a shard as following:
```
<headless>: join app
```
> This step is optional, an instance can be run without joining any shard, as a headless node. However, on that instance no transaction can be submitted. It will simply participate in the network as a headless node.

### Send counter transaction
Use the application's CLI to submit transactions as following:
```
<app>: incr cntr1 
adding transaction: incr cntr1 1

<app>: decr cntr2
adding transaction: decr cntr2 1
```
Use application's CLI to query a counter's value:
```
<app>: countr cntr1
     cntr1: 1
<app>: countr cntr2
     cntr2: -1
<app>: countr cntr3
     cntr3: not found
```

### Shutdown application
```
<app>: quit
Shutdown cleanly
```
## Release Notes
### Iteration 2
* different nodes can run different applications, and can even run without application (headless node)
* shard: app transaction filtering based on shard membership, supports headless node
* demo app: a simple network counter app that can register specific shard with DLT stack and send increment and decrement counter op transactions once all nodes are setup and connected (i.e. all nodes online at genesis). All application nodes within same shard should be consistent in counter values

### Iteration 1
* all app nodes are initialized and connected to the network before first transaction is submitted (node's do not join an existing network with previous history)
* nodes stay online and connected to the network for duration/life of the application/experiment
nodes do not need to persist and restore transaction history or consensus state
* all nodes are completely reliable (no fault tolerance required) and honest (no competing transactions are submitted)
* all nodes in the network are exactly identical and support one single application
* p2p: discovery and connection, basic handshake for protocol version compatibility
* endorsement: pass through, no anchoring and submission, no peer sync (all nodes online at genesis)
* shard: pass through (no app filtering, no shard membership, no headless node)
* privacy: pass through (no payload encryption/decryption/exchange)
* consensus: basic handshake for app version compatibility (should this actually be in shard layer?), no peer sync (all nodes online at genesis), no conflict/concurrency resolution, no persistence
* demo app: a simple network counter app that sends increment and decrement counter op transactions once all nodes are setup and connected (i.e. all nodes online at genesis)
