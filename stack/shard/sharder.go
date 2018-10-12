// Copyright 2018 The trust-net Authors
// Sharding Layer interface and implementation for DLT Statck
package shard

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
)

// application handshake message
type AppConfig struct {
	// public ID of the application instance (different from node ID used in p2p layer)
	AppId []byte
	// name of the application
	Name string
	// shard ID of the application (same for all nodes of application)
	ShardId []byte
}

// transaction message
type Transaction struct {
	// serialized transaction payload
	Payload []byte
	// transaction signature
	Signature []byte
	// transaction approver application instance ID
	AppId []byte
	// transaction submitter's public ID
	Submitter []byte
}

type Sharder interface {
	// register application shard with the DLT stack
	Register(shardId []byte, name string, txHandler func (tx *Transaction) error) error
	// unregister application shard from DLT stack
	Unregister() error
}

type sharder struct {
	db db.Database
}

func (s *sharder) Register(shardId []byte, name string, txHandler func (tx *Transaction) error) error {
	return errors.New("not yet implemented")
}

func (s *sharder) Unregister() error {
	return errors.New("not yet implemented")
}

func NewSharder(db db.Database) (*sharder, error) {
	return &sharder{
		db: db,
	}, nil
}