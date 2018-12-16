package dto

import (
	"github.com/trust-net/go-trust-net/common"
)

// transaction message
type Anchor struct {
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
}

func (a *Anchor) Serialize() ([]byte, error) {
	return common.Serialize(a)
}

func (a *Anchor) DeSerialize(data []byte) error {
	if err := common.Deserialize(data, a); err != nil {
		return err
	}
	return nil
}
