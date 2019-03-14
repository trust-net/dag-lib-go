// Copyright 2018-2019 The trust-net Authors
// An anchor for transaction meta data
package dto

import (
	"github.com/trust-net/dag-lib-go/common"
	"fmt"
)

// transaction message
type Anchor struct {
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
}

func (a *Anchor) ToString() string {
	return fmt.Sprintf("NodeId: %x\nShardSeq: %d, Weight: %d, ShardUncles: %d\nShardParent: %x\nSignature: %x",
		a.NodeId, a.ShardSeq, a.Weight, len(a.ShardUncles), a.ShardParent, a.Signature)
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

// we want to make sure we always create byte array for signature in a well known order
func (a *Anchor) Bytes() []byte {
	payload := make([]byte, 0, 1024)
	payload = append(payload, a.NodeId...)
	payload = append(payload, common.Uint64ToBytes(a.ShardSeq)...)
	payload = append(payload, common.Uint64ToBytes(a.Weight)...)
	payload = append(payload, a.ShardParent[:]...)
	for _, uncle := range a.ShardUncles {
		payload = append(payload, uncle[:]...)
	}
	return payload
}
