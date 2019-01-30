# dag-lib-go
Go library for [DAG protocol](https://github.com/trust-net/dag-documentation#dag-documentation)

* [Introduction](https://github.com/trust-net/dag-lib-go#Introduction)
* [How to setup workspace](https://github.com/trust-net/dag-lib-go#how-to-setup-workspace)
    * [Clone Repo](https://github.com/trust-net/dag-lib-go#clone-repo)
    * [Install Dependencies](https://github.com/trust-net/dag-lib-go#Install-Dependencies)
* [Example Applications Using DLT Stack Library](https://github.com/trust-net/dag-lib-go#Example-Applications-Using-DLT-Stack-Library)
    * [Build test applications](https://github.com/trust-net/dag-lib-go#Build-test-applications)
    * [Stage test applications](https://github.com/trust-net/dag-lib-go#Stage-test-applications)
    * [Create sub-directories for each instance of the test application](https://github.com/trust-net/dag-lib-go#Create-sub-directories-for-each-instance-of-the-test-application)
    * [Create config file for each instance](https://github.com/trust-net/dag-lib-go#Create-config-file-for-each-instance)
    * [Run the test applications](https://github.com/trust-net/dag-lib-go#Run-the-test-applications)
    * [Double Spender Application CLI](https://github.com/trust-net/dag-lib-go#Double-Spender-Application-CLI)
    * [Network Counter Application CLI](https://github.com/trust-net/dag-lib-go#Network-Counter-Application-CLI)
* [How to use DLT stack library in your application](https://github.com/trust-net/dag-lib-go#how-to-use-dlt-stack-library-in-your-application)
    * [Create configuration](https://github.com/trust-net/dag-lib-go#Create-configuration)
    * [Instantiate DLT stack](https://github.com/trust-net/dag-lib-go#Instantiate-DLT-stack)
    * [Register application with DLT stack](https://github.com/trust-net/dag-lib-go#Register-application-with-DLT-stack)
    * [Start the DLT stack](https://github.com/trust-net/dag-lib-go#Start-the-DLT-stack)
    * [Process transactions from network peers](https://github.com/trust-net/dag-lib-go#Process-transactions-from-network-peers)
    * [Stop DLT Stack](https://github.com/trust-net/dag-lib-go#Stop-DLT-Stack)
* [Release Notes](https://github.com/trust-net/dag-lib-go#Release-Notes)
    * [Iteration 7](#Iteration-7)
    * [Iteration 6](https://github.com/trust-net/dag-lib-go#Iteration-6)
    * [Iteration 5](https://github.com/trust-net/dag-lib-go#Iteration-5)
    * [Iteration 4](https://github.com/trust-net/dag-lib-go#Iteration-4)
    * [Iteration 3](https://github.com/trust-net/dag-lib-go#Iteration-3)
    * [Iteration 2](https://github.com/trust-net/dag-lib-go#Iteration-2)
    * [Iteration 1](https://github.com/trust-net/dag-lib-go#Iteration-1)

## Introduction
This is a golang based implementation of the DAG protocol stack, intended to build native golang applications with DLT support. Documentation for the actual protocol design and rules is [here](https://github.com/trust-net/dag-documentation#dag-documentation), wheras below documentation focuses on how to use the protocol stack library in an application.

## How to setup workspace

### Clone Repo
```
mkdir -p $GOPATH/src/trust-net
git clone git@github.com:trust-net/dag-lib-go.git
```

### Install Dependencies
Project uses Ethereum's `go-ethereum` for low level p2p and crypto libraries from `release/1.7	` branch. Install these dependencies as following:

```
mkdir -p $GOPATH/src/ethereum
cd $GOPATH/src/ethereum
git clone --single-branch --branch release/1.7  https://github.com/ethereum/go-ethereum.git 
```

After above step, install remaining dependencies using `go get` as following:

```
cd $GOPATH/src/trust-net/dag-lib-go/stack
go get
```

Above will install remaining dependencies into go workspace.

> Note: Ethereum dependency requires gcc/CC installed for compiling and building crypto library. Hence, `go get` may fail if gcc/CC is not found. Install the platform appropriate compiler and then re-run `go get`.


## Example Applications Using DLT Stack Library
Two test driver applications are provided to show examples and test different capabilities of DLT stack:

* `tests/countr/app.go`: A test program that implements a distributed network counter across different shards using DLT stack library
* `tests/spendr/app.go`: A test program that implements a value transfer application to test double spending resolution across multiple nodes in the network

### Build test applications

```
cd $GOPATH/src/trust-net/dag-lib-go/
(cd tests/countr/; go build)
(cd tests/spendr/; go build)
```

### Stage test applications
Copy the built binaries into a staging area, e.g.:

```
mkdir -p $USER/tmp/test-trust-node
cp $GOPATH/src/trust-net/dag-lib-go/tests/countr/countr $USER/tmp/test-trust-node
cp $GOPATH/src/trust-net/dag-lib-go/tests/spendr/spendr $USER/tmp/test-trust-node
```

### Create sub-directories for each instance of the test application
If planning to run multiple instances of the test application on same machine, create one sub-directory for each instance to host keys and config for that instance, e.g.:

```
cd $USER/tmp/test-trust-node
mkdir -p node-1 node-2 node-3 node-4
```

### Create config file for each instance
Copy the following example config files into sub-directory for each instance that need to be run:

```
{
	"key_file": "node.key",
	"key_type": "ECDSA_S256",
	"max_peers": 2,
	"node_name": "test node 1",
	"listen_port": "50883",
	"boot_nodes": ["enode://c3da24ed70538b731b9734e4e0b8206e441089ab4fcd1d0faadb1031e736491b70de0b70e1d581958b28eb43444491b3b9091bd8a81d1767bf7d4ebc3e7bd108@127.0.0.1:50884"]
}
```

> Please make sure:
* `listen_port` is unique in each copy
* `boot_nodes` has port from some other instance (e.g. node-1 listening on port 50883 will use node-2's listening port 50884)

### Run the test application
Start an instance of _any one_ of the two test applications with its corresponding config file (below example assumes config file for each instance is placed under its node-N directory and named as config.json):

```
### on terminal 1 run an instance of countr app
cd $USER/tmp/test-trust-node/node-1
$USER/tmp/test-trust-node/countr -config config.json

### on terminal 2 run an instance of spendr app
cd $USER/tmp/test-trust-node/node-2
$USER/tmp/test-trust-node/spendr -config config.json

### on terminal 3 run an instance of countr app
cd $USER/tmp/test-trust-node/node-3
$USER/tmp/test-trust-node/countr -config config.json

### on terminal 4 run an instance of spendr app
cd $USER/tmp/test-trust-node/node-4
$USER/tmp/test-trust-node/spendr -config config.json
```

> As per iteration 4, nodes can join/leave/re-join network dynamically while rest of the network is processing transactions. Nodes will perform on-demand sync at shard level during peer handshake, app registration and unknown transaction processing. Also, while nodes do not yet persist state across reboot, they perform full re-sync across reboot. Hence, as long as there is 1 active node on the network, progress will be made.

### Double Spender Application CLI
A test driver application is provided to demonstrate and validate the double spending resolution protocol of the DLT stack protocol. Application implements following capabilities:
* a simple "value transfer" functionality to demonstrate how such applications can be implemented using DLT stack
* support to simulate a "dishonest" client that submits double spending transaction on same node
* support to simulate a "dishonest" client that submits double spending transaction on two different nodes across the network


Refer to documentation for double spender application CLI at [documentation link](https://github.com/trust-net/dag-lib-go/issues/36)

### Network Counter Application CLI
A test driver application is provided that implements simple network counters isolated within specific name space, to demonstrate and validate support for multiple shards with proper isolation and sync across the shared network.

Refer to documentation for double spender application CLI at [documentation link](https://github.com/trust-net/dag-lib-go/issues/39)

## How to use DLT stack library in your application

### Create configuration
Create a `p2p.Config` instance. These values can be read from a file, using the field names as specified in the `json` directive against each of them. A sample file is:

```
{
	"key_file": "<name of file to persist private key for the node>",
	"key_type": "ECDSA_S256",
	"max_peers": <max number of peers to connect>,
	"node_name": "<name for the node instance>",
	"listen_port": "<unique port for the node instance>",
	"boot_nodes": ["<enode full URL of peers to connect with during boot up>"]
}
```

### Instantiate DLT stack
Use `stack.NewDltStack(conf p2p.Config, db db.Database)` method to instantiate a DLT stack controller. This takes two arguments:
* a `p2p.Config` structure with parameters as described above
* an implementation of `db.DbProvider` implementation, that will be used by DLT stack to instantiate DLT DB to save/retrieve/persist data

### Register application with DLT stack
If running an application on the DLT stack, then register the application with the DLT stack using the `stack.DLT.Register(shardId []byte, name string, txHandler func(tx dto.Transaction, state state.State) error) error` method. This takes following arguments:
* `shardId`: a byte array with unique identifier for the shard of the application
* `name`: a name of the application
* `txHandler`: a function that will be called to accept/process a transaction from a peer and update the world state. If this method returns back a non-nil error, then transaction will not be forwarded to other network peers and world state updates will be discarded

> This step is optional because a deployment may choose to run in "headless" mode, in which case it will not process any application transactions and will only participate in the transaction endorsement process, to provide network security.

### Start the DLT stack
Use `stack.DLT.Start()` method for DLT stack to start discovering other nodes on the network and start listening to messages from connected peers. 

### Process transactions from network peers
If application had registered with DLT stack with appropriate callback methods, then after DLT stack is started, whenever a new network transaction is received, the application provided "`func(tx dto.Transaction, state state.State) error`" implementation is called with transaction details and a reference to shard's world state. Application is suppose to return back an error if transaction was not accepted.

### Stop DLT Stack
Once application execution completes (either due to application shutdown, or any other reason), call the `stack.DLT.Stop()` method to disconnect from all connected network peers.

## Release Notes

### Iteration 7
* Made protocol's hashing and encryption compatible with ethereumJ based [Java client](https://github.com/trust-net/dag-lib-java)
* Extended Spendr test application to support [REST API](https://github.com/trust-net/dag-lib-go/issues/46) for a submitter/client access
* Built a companion [Android App](https://github.com/trust-net/SpendrClient) as sample submitter/client to work with Spendr's REST API
* Details of the changes are captured in the [Iteration 7](https://github.com/trust-net/dag-lib-go/issues/45).

### Iteration 6
Details of the changes are captured in the [Iteration 6](https://github.com/trust-net/dag-lib-go/issues/42).

### Iteration 5

**Assumptions**:

* All submitters are required to maintain correct latest transaction sequence and transaction ID for submitted transactions by them
* A submitter's transaction sequence will be a monotonically increasing number for all transactions by that submitter across all different application shards that submitter submits transactions to
* A new transaction will have reference to previous transaction sequence and ID from the submitter
* There are 2 types of transactions defined for any application: "outgoing" transactions where some value gets transferred out from assets belonging to a network identity, and "incoming" transactions where some value gets transferred into assets belonging to a network identity. Only asset owners can submit "outgoing" transactions against an asset, whereas anyone can submit an "incoming" transaction towards an asset.

**Implementation**:

* DLT stack validates a locally submitted transaction for correct sequence number from submitter as following:
  * if submitted transaction from same submitter, sequence number and shard already exists, then submitted transaction will be rejected as double spend
  * else (i.e., there is no other transaction from same submitter, using same submitter sequence, for the same shard) then
    * if parent transaction from same submitter and previous sequence number does not exists, then submitted transaction will be rejected as out of sequence
    * if parent transaction from same submitter and previous sequence number does not match submitted transaction's parent, then submitted transaction will be rejected as invalid
    * else submitted transaction will be accepted and broadcasted to network
* Similarly, a network transaction is validated as following:
  * if node's submitter history has an existing transaction that has same submitter ID, Submitter Sequence and Shard as the received network transaction, then
    * if the transaction ID in local submitter history is same as received transaction, then we discard received transaction as duplicate
    * else (i.e. network transaction ID is different from local transaction) trigger a **double spending resolution**
  * else (i.e., there is no other transaction from same submitter, using same submitter sequence, for the same shard) then
  * local submitter history does not have the transaction mentioned in "Previous Submitter Tx" of the received network transaction, then trigger a submitter history sync with the peer node
  * else (i.e. local history has the reference last submitter transaction) accept the network transaction for processing
* A **consensus** protocol is defined, to resolve double spending when 2 competing transactions from same submitter and sequence are seen
  * protocol is able to handle the impact to shard DAG, since transactions cannot be rolled back in sharding DAG
  * protocol is able to handle when a competing/duplicate network transaction is seen after a submission is accepted from client
  * protocol is able to handle when two competing network transactions from different peers are seen on a node
* A **submitter sync** protocol is defined for peer's to update submitter transaction sequences for out of sync network transactions.

### Iteration 4
* nodes can join the network at any time after transaction processing has started
* app can shutdown/leave the network and join back dynamically (does not need to be running since the beginning)
* DLT stack will perform on-demand shard layer sync upon peer handshake, app registration and unknown transaction processing

### Iteration 3
* app can join and leave shards dynamically (node needs to continue running since beginning, however app's registration to different shards is dynamic)
* app: will get full sync for shard's transactions upon joining a new shard, or when it leaves then re-joins same shard
* client: an application's transaction needs to be signed by the client when submitting to DLT stack
* demo app: extended CLI to join/leave different shards dynamically and submit signed transactions to the DLT stack

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
