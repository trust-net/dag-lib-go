## Spendr: A Value Transfer Application

* [Application Architecture](#Application-Architecture)
    * [Application Shard Specs](#Application-Shard-Specs)
    * [Application Transactions](#Application-Transactions)
    * [Multiple Nodes Support](#Multiple-Nodes-Support)
* [Transaction Ops](#Transaction-Ops)
    * [Resource Creation Op](#Resource-Creation-Op)
    * [Value Transfer Op](#Value-Transfer-Op)
* [CLI Commands](#CLI-Commands)
    * [Local Double Spend](#Local-Double-Spend)
    * [Multiple Node Submission](#Multiple-Node-Submission)
    * [Split Shard Submission](#Split-Shard-Submission)
* [REST Endpoints](#REST-Endpoints)
    * [Get Resource](#Get-Resource)
    * [Resource Creation Payload](#Resource-Creation-Payload)
    * [Transfer Value Payload](#Transfer-Value-Payload)
    * [Submit Transaction](#Submit-Transaction)

## Application Architecture
A test driver application is provided to demonstrate and validate the double spending resolution protocol of the DLT stack protocol. Application implements following capabilities:
* a simple "value transfer" functionality to demonstrate how such applications can be implemented using DLT stack
* support to simulate a "dishonest" client that submits double spending transaction on same node
* support to simulate a "dishonest" client that submits double spending transaction on two different nodes across the network
* a REST API endpoint for submitting transaction requests by remove clients

### Application Shard Specs
We'll run the test driver application on the shared trust-net network, where all other application (e.g. network counter) are running. This test driver application will register with DLT stack using following specs:
* shard id: `[]byte("test-driver-for-double-spending")`
* name: `"test-driver-for-double-spending"`


### Application Transactions
Test driver will implement "ops" that will be submitted via transaction and processed by the application's registered transaction handlers. Following Ops specs will be used to submit as serialized and signed payload:

```
type Ops struct {
  // op code
  Code uint64
  // serialized arguments
  Args []byte
}
```
In above structure, application will define a unique value for `Code` for each specific operation defined below. Also, application will define a corresponding structure of arguments that will be stored as  a serialized blob with `Args`.

### Multiple Nodes Support
In order to submit double spending transactions across different nodes, we'll extend the test driver application to instantiate multiple DLT stacks listening on different ports. This way the CLI can submit transactions to those different instances of the stack, each of which is effectively a separate application instance node.

## Transaction Ops
Application defined following operations that can be requested by clients/submitters...

### Resource Creation Op
Driver will provide CLI to create a resource with initial value:

```
CLI> create <resource name> <initial value>
```
Above command will submit an `Ops` with following values:
#### `Code` value

`0x01`

#### `Args` structure

```
type ArgsCreate struct {
  // resource name
  Name string
  // initial value
  Value  int64
}
```

#### Rules of processing
* there must not be a resource with that same name present
* resource will the created with submitter's ID as owner of the resource
* initial value must be an integer (for simplicity we are not dealing with fractions)

### Value Transfer Op
Driver will provide CLI to transfer value from owned resource to another resource:

```
CLI> xfer <owned resource name> <xfer value> <recipient resource name>
```
Above command will submit an `Ops` with following values:

#### `Code` value

`0x02`

#### `Args` structure

```
type ArgsXfer struct {
  // xfer source name
  Source string
  // xfer destination name
  Destination string
  // xfer value
  Value  int64
}
```

#### Rules of processing
* source of transfer resource must be owned by the submitter
* xfer value cannot be greater that the current value for resource
* xfer value cannot be less that `1` (i.e. cannot be zero or negative)
* xfer value must be a positive integer (for simplicity we are not dealing with fractions)
* operation will result in xfer value deducted from source resource and added to destination resource

## CLI Commands
Application provides following CLI commands for testing different double spending scenarios...

> Each application instance will create a private instance of a submitter client for its CLI. This submitter instance may get corrupted when attempting double spending transactions below, since network may converge on a "Last Transaction" from the submitter that is different from what submitter is tracking. This corruption is acceptable "punishment" due to intentional double spending attempt of the submitter.

### Local Double Spend
Test driver will implement a new CLI command that can be used to submit two double spending transactions as following:

```
CLI> double <owned resource name> <xfer value> <recipient 1 resource name> <recipient 2 resource name>
```
Above command will submit 2 value transfer transactions in succession (i.e., sequential shard seq) using the same submitter seq on both anchors.

### Multiple Node Submission
Test driver will implement a new CLI command that can be used to submit a redundant transactions on two different nodes, so that even if one fails there is a redundancy and another submission passes.

```
CLI> multi <owned resource name> <xfer value> <recipient resource name>
```
Above command will submit same value transfer operation in parallel to two different nodes using the same transaction request. Network should only accept 1 of the two transactions, while the other transaction should be rejected as double spending.

### Split Shard Submission
Test driver will implement a new CLI command that can be used to submit two double spending transaction requests on two different nodes to transfer same value to different recipients. Goal is to try and split the shard across network by submitting conflicting/double spending transaction on different nodes.

```
CLI> usage: split <owned resource name> <xfer value> <recipient 1> <recipient 2>
```
Above command will submit 2 different value transfer operations as 2 separate transaction requests in parallel to two different nodes using the same submitter seq. Network should eventually detect this double spending attempt and consistently converge on only one of the value transfer operation while rejecting the other operation on all nodes. World state on all nodes should show consistent and correct values for all recipients.

## REST Endpoints
### Get Resource
tbd
### Resource Creation Payload
tbd
### Transfer Value Payload
tbd
### Submit Transaction
tbd