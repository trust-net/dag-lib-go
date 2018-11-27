// Copyright 2018 The trust-net Authors
// Sharding Layer interface and implementation for DLT Statck
package shard

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
)

type Sharder interface {
	// register application shard with the DLT stack
	Register(shardId []byte, txHandler func(tx *dto.Transaction) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// Handle Transaction
	Handle(tx *dto.Transaction) error
}

type sharder struct {
	db db.Database

	shardId   []byte
	txHandler func(tx *dto.Transaction) error
}

func (s *sharder) Register(shardId []byte, txHandler func(tx *dto.Transaction) error) error {
	s.shardId = append(shardId)
	s.txHandler = txHandler
	return nil
}

func (s *sharder) Unregister() error {
	s.shardId = nil
	s.txHandler = nil
	return nil
}

func (s *sharder) Handle(tx *dto.Transaction) error {
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
