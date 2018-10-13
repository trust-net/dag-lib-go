// Copyright 2018 The trust-net Authors
// Sharding Layer interface and implementation for DLT Statck
package shard

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
)

// transaction message
type Transaction struct {
	// serialized transaction payload
	Payload []byte
	// transaction signature
	Signature []byte
	// transaction approver application instance ID
	AppId []byte
	// transaction approver application's shard ID
	ShardId []byte
	// transaction submitter's public ID
	Submitter []byte
}

type Sharder interface {
	// register application shard with the DLT stack
	Register(shardId []byte, txHandler func(tx *Transaction) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// Handle Transaction
	Handle(tx *Transaction) error
}

type sharder struct {
	db db.Database

	shardId   []byte
	txHandler func(tx *Transaction) error
}

func (s *sharder) Register(shardId []byte, txHandler func(tx *Transaction) error) error {
	s.shardId = append(shardId)
	s.txHandler = txHandler
	return nil
}

func (s *sharder) Unregister() error {
	s.shardId = nil
	s.txHandler = nil
	return nil
}

func (s *sharder) Handle(tx *Transaction) error {
	// validate transaction
	if len(tx.ShardId) == 0 {
		return errors.New("missing shard id in transaction")
	}
	if s.txHandler != nil {
		if string(s.shardId) == string(tx.ShardId) {
			return s.txHandler(tx)
		}
	}
	return nil

}

func NewSharder(db db.Database) (*sharder, error) {
	return &sharder{
		db: db,
	}, nil
}
