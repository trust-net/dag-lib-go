// Copyright 2018 The trust-net Authors
// Common DTO types used throughout DLT stack
package dto

import (
	"crypto/sha512"
	"github.com/trust-net/go-trust-net/common"
)

// transaction message
type Transaction struct {
	// id of the transaction created from its hash
	id     [64]byte
	idDone bool
	// serialized transaction payload
	Payload []byte
	// transaction signature
	Signature []byte
	// transaction approver application instance node ID
	NodeId []byte
	// transaction approver application's shard ID
	ShardId []byte
	// sequence of this transaction within the shard
	ShardSeq []byte
	// parent transaction within the shard
	ShardParent []byte
	// transaction submitter's public ID
	Submitter []byte
}

// compute SHA512 hash or return from cache
func (tx *Transaction) Id() []byte {
	if tx.idDone {
		return tx.id[:]
	}
	data := make([]byte, 0)
	// signature should be sufficient to capture payload and submitter ID
	data = append(data, tx.Signature...)
	// append shard ID etc
	// TBD: replace with TxAnchor when available
	data = append(data, tx.ShardId...)
	data = append(data, tx.ShardSeq...)
	data = append(data, tx.ShardParent...)
	tx.id = sha512.Sum512(data)
	tx.idDone = true
	return tx.id[:]
}

func (tx *Transaction) Serialize() ([]byte, error) {
	return common.Serialize(tx)
}

func (tx *Transaction) DeSerialize(data []byte) error {
	if err := common.Deserialize(data, tx); err != nil {
		return err
	}
	return nil
}
