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
* [API Specifications](#API-Specifications)
    * [Op: Query Resource Value](#Op-Query-Resource-Value)
    * [Op: Resource Creation Payload](#Op-Resource-Creation-Payload)
    * [Op: Value Transfer Payload](#Op-Value-Transfer-Payload)
    * [Op: Submit Transaction](#Op-Submit-Transaction)

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

## API Specifications
Application provides REST API to perform following operations from a remote client:
* Create Resource
  * Fetch the payload for "Create Resource" operation from Spendr's `Op: Resource Creation Payload`
  * Create a signed transaction request using provided payload and submitter's meta data
  * Submit transaction request using Spendr's `Op: Submit Transaction`
* Value Transfer
  * Fetch the payload for "Value Transfer" operation from Spendr's `Op: Value Transfer Payload`
  * Create a signed transaction request using provided payload and submitter's meta data
  * Submit transaction request using Spendr's `Op: Submit Transaction`

### Op: Query Resource Value
This is a simple GET of specific resource by its key:

```
GET /resources/{key}
```
And response would consist of resource information as following:

```
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Resource",
  "description": "A world state resource for spendr application",
  "type": "object",
  "properties": {
    "key": {
      "description": "The unique identifier for a resource within spendr application",
      "type": "string"
    },
    "owner": {
      "description": "130 char hex encoded identity of the owner of resource",
      "type": "string"
    },
    "value": {
      "description": "64 bit unsigned integer value for resource at the time of query",
      "type": "integer"
    }
  },
  "required": [ "key", "owner", "value" ]
}
```

### Op: Resource Creation Payload
Request payload as per application's syntax/semantics for transaction payloads, for creating a new resource:

```
POST /opCode/create

{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Resource",
  "description": "A world state resource for spendr application",
  "type": "object",
  "properties": {
    "key": {
      "description": "The unique identifier for a resource within spendr application",
      "type": "string"
    },
    "value": {
      "description": "64 bit unsigned integer value for resource at creation",
      "type": "integer"
    }
  },
  "required": [ "key", "value" ]
}
```

<a id="#opcode"></a>Successful response for a payload request will return following `OpCode` response:

```
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "OpCode",
  "description": "application's opcode for requested operation",
  "type": "object",
  "properties": {
    "payload": {
      "description": "a base64 encoded string for serialized (and optionally encrypted) payload as per application syntax/semantics for requested operation",
      "type": "string"
    },
    "description": {
      "description": "a user friendly description of the payload operation",
      "type": "string"
    }
  },
  "required": [ "payload", "description"]
}
```

### Op: Value Transfer Payload
Request payload as per application's syntax/semantics for transaction payloads, for transferring value from owned resource to another resource:

```
POST /opcode/xfer

{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "OpCodeXferValueRequest",
  "description": "A spendr application OpCode request to transfer value from owned resource",
  "type": "object",
  "properties": {
    "source": {
      "description": "The unique identifier for an owned resource within spendr application",
      "type": "string"
    },
    "destination": {
      "description": "The unique identifier for destination resource within spendr application",
      "type": "string"
    },
    "value": {
      "description": "64 bit unsigned integer value to transfer from owned resource to destination resource",
      "type": "integer"
    }
  },
  "required": [ "source", "destination", "value" ]
}
```
Successful response for above will return the _**OpCode**_ as defined [above](#opcode).

### Op: Submit Transaction
Submit a new application transaction request using submitter's history and payload provided by application:

```
POST /transactions

{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SubmitRequest",
  "description": "Request to submit a new transaction",
  "type": "object",
  "properties": {
    "payload": {
      "description": "a base64 encoded payload for application",
      "type": "string"
    },
    "shard_id": {
      "description": "a hex encoded string uniquely identifying the shard of the application",
      "type": "string"
    },
    "submitter_id": {
      "description": "130 char hex encoded public id of the submitter",
      "type": "string"
    },
    "last_tx": {
      "description": "130 char hex encoded id of the last transaction from submitter",
      "type": "string"
    },
    "submitter_seq": {
      "description": "64 bit unsigned integer value for transaction sequence from submitter",
      "type": "integer"
    },
    "padding": {
      "description": "64 bit unsigned integer value for padding request hash for PoW",
      "type": "integer"
    },
    "signature": {
      "description": "a base64 encoded ECDSA secpk256 signature of request using private key of submitter",
      "type": "string"
    }
  },
  "required": [ "payload", "shard_id", "submitter_id", "last_tx", "submitter_seq", "padding", "signature" ]
}
```

Above request should return a success response for transaction submission as following:

```
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SubmitResponse",
  "description": "Response to successful transaction submission",
  "type": "object",
  "properties": {
    "tx_id": {
      "description": "130 char hex encoded id of the submitted transaction",
      "type": "string"
    }
  },
  "required": [ "tx_id"]
}
```
