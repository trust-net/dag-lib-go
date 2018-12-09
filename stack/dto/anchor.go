package dto

import ()

// transaction message
type Anchor struct {
	// transaction approver application instance node ID
	NodeId []byte
	// transaction approver application's shard ID
	ShardId []byte
	// sequence of this transaction within the shard
	ShardSeq uint64
	// parent transaction within the shard
	ShardParent [64]byte
}
