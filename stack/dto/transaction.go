// Copyright 2018 The trust-net Authors
// Common DTO types used throughout DLT stack
package dto

import (
	"crypto/sha512"
	//	"encoding/binary"
	"github.com/trust-net/go-trust-net/common"
)

type Transaction interface {
	Id() [64]byte
	Serialize() ([]byte, error)
	DeSerialize(data []byte) error
	Anchor() *Anchor
	Self() *transaction
}

// transaction message
type transaction struct {
	// id of the transaction created from its hash
	id     [64]byte
	idDone bool
	// serialized transaction payload
	Payload []byte
	// transaction payload signature by the submitter
	Signature []byte
	// transaction anchor from DLT stack
	TxAnchor *Anchor
	// anchor signature from DLT stack
	AnchorSignature []byte
	//	// transaction approver application instance node ID
	//	NodeId []byte
	//	// transaction approver application's shard ID
	//	ShardId []byte
	//	// sequence of this transaction within the shard
	//	ShardSeq uint64
	//	// parent transaction within the shard
	//	ShardParent [64]byte
	//	// uncle transactions withing the shard
	//	ShardUncles [][64]byte
	//	// transaction submitter's public ID
	//	Submitter []byte
}

// compute SHA512 hash or return from cache
func (tx *transaction) Id() [64]byte {
	if tx.idDone {
		return tx.id
	}
	data := make([]byte, 0)
	// signature should be sufficient to capture payload and submitter ID
	data = append(data, tx.Signature...)
	// append anchor's signature
	data = append(data, tx.AnchorSignature...)
	//	// append shard ID etc
	//	data = append(data, tx.ShardId...)
	//	seq := [8]byte{}
	//	binary.BigEndian.PutUint64(seq[:], tx.ShardSeq)
	//	data = append(data, seq[:]...)
	//	data = append(data, tx.ShardParent[:]...)
	tx.id = sha512.Sum512(data)
	tx.idDone = true
	return tx.id
}

func (tx *transaction) Serialize() ([]byte, error) {
	return common.Serialize(tx)
}

func (tx *transaction) DeSerialize(data []byte) error {
	if err := common.Deserialize(data, tx); err != nil {
		return err
	}
	return nil
}

func (tx *transaction) Anchor() *Anchor {
	return tx.TxAnchor
}

func (tx *transaction) Self() *transaction {
	return tx
}

// make sure any Transaction can only be created with an anchor
func NewTransaction(a *Anchor) *transaction {
	if a == nil {
		return nil
	}
	return &transaction{
		TxAnchor: a,
	}
}
