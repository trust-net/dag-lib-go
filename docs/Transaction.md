## Contents

* [Transaction Schema](#Transaction-Schema)
    * [TxRequest](#TxRequest)
    * [TxAnchor](#TxAnchor)
    * [Transaction ID](#Transaction-ID)
* [Transaction Request Schema](#Transaction-Request-Schema)
    * [Payload](#Payload)
    * [Shard Id](#Shard-Id)
    * [Submitter Id](#Submitter-Id)
    * [Last Transaction](#Last-Transaction)
    * [Submitter Sequence](#Submitter-Sequence)
    * [Padding](#Padding)
    * [Request Signature](#Request-Signature)
* [Anchor Schema](#Anchor-Schema)
    * [Node Id](#Node-Id)
    * [Shard Sequence](#Shard-Sequence)
    * [Shard Parent](#Shard-Parent)
    * [Shard Uncles](#Shard-Uncles)
    * [Weight](#Weight)
    * [Anchor Signature](#Anchor-Signature)

## Transaction Schema
A transaction consists of the following fields:

```
// transaction request from submitter
TxRequest *TxRequest
// transaction anchor from DLT stack
TxAnchor *Anchor
```
### TxRequest
`TxRequest` is the transaction request from a submitter. It contains meta data necessary to order transaction correctly in right submitter's history sequence.

### TxAnchor
The `TxAnchor` is a signed meta data about the transaction's place in the shard DAG, as seen by the transaction approving application instance node. It contains meta data necessary to order transaction correctly in application's shard DAG.

### Transaction ID
The transaction's ID is a `[64]byte` unique identifier computed from a SHA512 hash of transaction's contents as following:

```
data := make([]byte, 0)
// start with submitter's request signature
data = append(data, tx.Request().Signature...)
// append anchor's signature
data = append(data, tx.Anchor().Signature...)
tx.id = sha512.Sum512(data)

```

> Due to the trust-less nature of network, the transaction Id is not included with transaction bytes exchanged over the network, rather its computed locally by each node when it receives a transaction's bytes from a peer.


## Transaction Request Schema
A transaction request has following schema:

```
// payload for transaction's operations
Payload []byte
// shard id for the transaction
ShardId []byte
// Submitter's public ID
SubmitterId []byte
// submitter's last transaction
LastTx [64]byte
// submitter's transaction sequence
SubmitterSeq uint64
// a padding to meet challenge for network's DoS protection
Padding uint64
// signature of the transaction request's contents using submitter's private key
Signature []byte
```

### Payload
`Payload` is the serialized transaction data as per application/shard's schema. It is opaque to the DLT stack. Application instances of a shard determine and agree on their own schema and semantics for handling this data.

> This data element could be encrypted by application instances using an off-the-chain encryption mechanism.

### Shard Id
`ShardId` is the byte array uniquely identifying the application shard for an anchor (and its transaction).

### Submitter Id
`SubmitterId` is the ECDSA public key of the transaction requester. This value will be used in validating the signature of transation payload later. Also, protocol uses full length (65 bytes) of the public key as Id of submitter.

### Last Transaction
`LastTx` is the reference to last transaction from submitter's history, provided by the submitter as a parameter for transaction request. 

### Submitter Sequence
`SubmitterSeq` is a monotonically increasing number to track the number of transactions submitted by the transaction requestor. This value is also used as "nonce" for transaction payload signature, to protect against replay attacks. Also, this value is used for double spending detection, as described in more details with [DAG protocol documentation](https://github.com/trust-net/dag-documentation#Submitter-Sequencing-Rules).

### Padding
`Padding` is an unsigned 8 byte integer, used for meeting proof of work requirements by the network. This minimal PoW scheme is used as a deterrent against denial of service or spam attack by submitters.

> Current implementation does not enforce any PoW, so this is kept at 0x0 

### Request Signature
`Signature` is a 64 byte array of the ECDSA signature (`bytes[:32] == R, bytes[32:] == S`) using submitter's private key over SHA256 digest of transaction reauest parameters. Assuming that reference to transaction request is in `r`, then the signature can be calculated as following:

```
// create a place holder for request's bytes
bytes := make([]byte, 0, len(r.Payload)+len(r.ShardId)+145)

// start with payload of the request
bytes = append(bytes, r.Payload...)

// follow that with shard ID for this request
bytes = append(bytes, r.ShardId...)

// add submitter's last transaction ID
bytes = append(bytes, r.LastTx[:]...)

// then the submitter's public ID itself
bytes = append(bytes, r.SubmitterId...)

// add submiter's sequence for this transaction request
bytes = append(bytes, common.Uint64ToBytes(r.SubmitterSeq)...)

// finally the padding to meet PoW needs
bytes = append(bytes, common.Uint64ToBytes(r.Padding)...)

// compute SHA256 digest of the bytes
hash := sha256.Sum256(bytes)

// sign the hash using ECDSA private key of the submitter
type signature struct {
	R *big.Int
	S *big.Int
}
s := signature{}
s.R, s.S, _ = ecdsa.Sign(rand.Reader, submitterKey, hash[:])

// serialize the signature to [64]byte
serializedSignature := append(sig.R.Bytes(), sig.S.Bytes()...)
```

## Anchor Schema
Following are the contents of an Anchor in a transaction:

```
// transaction approver application instance node ID
NodeId []byte
// sequence of this transaction within the shard
ShardSeq uint64
// weight of this transaction withing shard DAG (sum of all ancestor's weight + 1)
Weight uint64
// parent transaction within the shard
ShardParent [64]byte
// uncle transactions within the shard
ShardUncles [][64]byte
// anchor signature from DLT stack
Signature []byte
```

### Node Id
`NodeId` is the public key (full length, 65 bytes) of the node's ECDSA key. It's included in the anchor to validate the anchor signature when a transaction is received/processed at each node in the network. Also, Protocol uses the full length of the key for node's ID.

### Shard Sequence
`ShardSeq` is the depth of shard's DAG at the time of anchor request.

### Shard Parent
`ShardParent` is one of the tips in shard's DAG that is selected by the node as ancestor to the anchor. Parent is selected as following:
* find all current tips from shard's DAG
* find the tips with greated depth (i.e. shard sequence)
* if there are multiple tips with same deepest value, then select the tip with highest numeric hash (i.e. numeric summation of all bytes of the transaction's Id)

### Shard Uncles
`ShardUncles` are all the remaining tips in shard's DAG after the parent tip is selected as `ShardParent`.

### Weight
`Weight` is the combined weight of all known tips of shard's DAG, at the time of anchor request, plus `1` for the new anchor. Essentially, this is a measure of how much value this new transaction anchor adds by cryptographically linking with other tips in the shard's DAG.

### Anchor Signature
The `Signature` in an anchor is a 64 byte array of the ECDSA signature (`bytes[:32] == R, bytes[32:] == S`) using node's private key over SHA256 digest of Anchor's parameters in order as following (assuming reference to anchor is in `a`):

```
// initialize a byte array to collect anchor parameters in order
payload := make([]byte, 0, 1024)

// append parameters from "a" in following order
payload = append(payload, a.NodeId...)
payload = append(payload, common.Uint64ToBytes(a.ShardSeq)...)
payload = append(payload, common.Uint64ToBytes(a.Weight)...)
payload = append(payload, a.ShardParent[:]...)
for _, uncle := range a.ShardUncles {
	payload = append(payload, uncle[:]...)
}

// compute SHA256 hash of the bytes
hash := sha256.Sum256(payload)

// sign the hash using ECDSA private key of the node
type signature struct {
	R *big.Int
	S *big.Int
}
s := signature{}
s.R, s.S, _ = ecdsa.Sign(rand.Reader, nodeKey, hash[:])

// serialize the signature to add to the anchor
a.Signature := append(s.R.Bytes(), s.S.Bytes()...)
```
