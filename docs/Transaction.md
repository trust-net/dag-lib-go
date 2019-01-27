## Contents

* [Transaction Schema](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Transaction-Schema)
    * [Payload](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Payload)
    * [Signature](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Signature)
    * [TxAnchor](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#TxAnchor)
    * [Transaction ID](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Transaction-ID)
* [Anchor Schema](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Anchor-Schema)
    * [Node Id](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Node-Id)
    * [Shard Id](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Shard-Id)
    * [Shard Sequence](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Shard-Sequence)
    * [Shard Parent](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Shard-Parent)
    * [Shard Uncles](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Shard-Uncles)
    * [Weight](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Weight)
    * [Submitter Id](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Submitter-Id)
    * [Submitter Sequence](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Submitter-Sequence)
    * [Submitter Last Transaction](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Submitter-Last-Transaction)
    * [Anchor Signature](https://github.com/trust-net/dag-lib-go/tree/master/docs/Transaction.md#Anchor-Signature)

## Transaction Schema
A transaction consists of the following fields:

```
// serialized transaction payload
Payload []byte
// transaction payload signature by the submitter
Signature []byte
// transaction anchor from DLT stack
TxAnchor *Anchor
```
### Payload
`Payload` is the serialized transaction data as per application/shard's schema. It is opaque to the DLT stack. Application instances for each shard determine their own schema and semantics for handling this data.


### Signature
`Signature` is a 64 byte array of the ECDSA signature (`bytes[:32] == R, bytes[32:] == S`) using submitter's private key over SHA256 digest of submitter's sequence and the transaction payload. Assuming that payload is serialized in a byte array `payload` and reference to transaction is in `tx`, then the signature can be calculated as following:

```
// use the submitter's sequence in transaction's anchor as nonce value
nonce := tx.Anchor().SubmitterSeq

// append the nonce value in bytes with the payload bytes
noncedBytes := append(common.Uint64ToBytes(nonce), payload...)

// compute SHA256 digest of the bytes
hash := sha256.Sum256(noncedBytes)

// sign the hash using ECDS private key of the submitter
type signature struct {
	R *big.Int
	S *big.Int
}
s := signature{}
s.R, s.S, _ = ecdsa.Sign(rand.Reader, submitterKey, hash[:])

// serialize the signature to [64]byte
serializedSignature := append(sig.R.Bytes(), sig.S.Bytes()...)
```

### TxAnchor
The `TxAnchor` is a signed artifact requested by the transaction submitter from an application instance node, and it contains proof of the transaction's meta data to help place transaction in right application shard and submitter's history sequence.

### Transaction ID
The transaction's ID is a `[64]byte` unique identifier computed from a SHA512 hash of transaction's contents as following:

```
data := make([]byte, 0)
// signature should be sufficient to capture payload and submitter ID
data = append(data, tx.Signature...)
// append anchor's signature
data = append(data, tx.TxAnchor.Signature...)
tx.id = sha512.Sum512(data)

```

> Due to the trust-less nature of network, the transaction Id is not included with transaction structure exchanged over the network, rather its computed locally by each node when it receives a transaction's contents from a peer.

## Anchor Schema
Following are the contents of an Anchor:

```
// transaction approver application instance node ID
NodeId []byte
// transaction approver application's shard ID
ShardId []byte
// sequence of this transaction within the shard
ShardSeq uint64
// weight of this transaction withing shard DAG (sum of all ancestor's weight + 1)
Weight uint64
// parent transaction within the shard
ShardParent [64]byte
// uncle transactions within the shard
ShardUncles [][64]byte
// transaction submitter's public ID
Submitter []byte
// submitter's last transaction ID
SubmitterLastTx [64]byte
// submitter's transaction sequence number
SubmitterSeq uint64
// anchor signature from DLT stack
Signature []byte
```

### Node Id
`NodeId` is the public key (full length) of the node's ECDSA key. It's included in the anchor to validate the anchor signature when a transaction is received/processed at each node in the network. Also, Protocol uses the full length of the key for node's ID.

### Shard Id
`ShardId` is the byte array uniquely identifying the application shard for an anchor (and its transaction).

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

### Submitter Id
`Submitter` is the ECDSA public key of the anchor requester. This value will be used in validating the signature of transation payload later. Also, protocol uses full length of the public key as Id of submitter.

### Submitter Sequence
`SubmitterSeq` is a monotonically increasing number to track the number of transactions submitted by the anchor requestor. This value is also used as "nonce" for transaction payload signature. Also, this value is used for double spending detection, as described in more details with [DAG protocol documentation](https://github.com/trust-net/dag-documentation#Submitter-Sequencing-Rules).

### Submitter Last Transaction
`SubmitterLastTx` is the reference to last transaction from submitter's history, provided by the submitter as a parameter for anchor request. 

### Anchor Signature
The `Signature` in an anchor is a 64 byte array of the ECDSA signature (`bytes[:32] == R, bytes[32:] == S`) using node's private key over SHA256 digest of Anchor's parameters in order as following:

```
// initialize a byte array to collect anchor parameters in order
payload := make([]byte, 0, 1024)

// append anchor's parameters in following order
payload = append(payload, a.ShardId...)
payload = append(payload, a.NodeId...)
payload = append(payload, a.Submitter...)
payload = append(payload, a.ShardParent[:]...)
for _, uncle := range a.ShardUncles {
	payload = append(payload, uncle[:]...)
}
payload = append(payload, common.Uint64ToBytes(a.ShardSeq)...)
payload = append(payload, common.Uint64ToBytes(a.Weight)...)
payload = append(payload, a.SubmitterLastTx[:]...)
payload = append(payload, common.Uint64ToBytes(a.SubmitterSeq)...)

// compute SHA512 hash of the bytes
hash := sha256.Sum256(payload)

// sign the hash using ECDS private key of the node
type signature struct {
	R *big.Int
	S *big.Int
}
s := signature{}
s.R, s.S, _ = ecdsa.Sign(rand.Reader, nodeKey, hash[:])

// serialize the signature to add to the anchor
serializedSignature := append(s.R.Bytes(), s.S.Bytes()...)
```
