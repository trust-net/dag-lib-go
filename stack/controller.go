// Copyright 2018 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"errors"
	"sync"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/p2p"
)

type AppConfig struct {
	// public ID of the application instance (different from node ID used in p2p layer)
	AppId []byte
	// name of the application
	Name string
	// shard ID of the application (same for all nodes of application)
	ShardId []byte
	// protocol version for the shard (same for all nodes of application)
	Version uint64
}

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

// approve if connection request is from a valid application peer
type PeerApprover func (app AppConfig) bool

// approve if a recieved network transaction is valid 
type NetworkTxApprover func (tx *Transaction) error

type DLT interface {
	// register application shard with the DLT stack
	Register(app AppConfig, peerHandler PeerApprover, txHandler NetworkTxApprover) error
	// unregister application shard from DLT stack
	Unregister() error
	// submit a transaction to the network
	Submit(tx *Transaction) error
	// start the controller
	Start() error
}

type dlt struct {
	app *AppConfig
	peerHandler PeerApprover
	txHandler NetworkTxApprover
	db db.Database
	lock   sync.RWMutex
}

func (d *dlt) Register(app AppConfig, peerHandler PeerApprover, txHandler NetworkTxApprover) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.app != nil {
		return errors.New("App is already registered")
	}
	d.app = &app
	d.peerHandler = peerHandler
	d.txHandler = txHandler
	return nil
}

func (d *dlt) Unregister() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.app = nil
	d.peerHandler = nil
	d.txHandler = nil
	return nil
}

func (d *dlt) Submit(tx *Transaction) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.app == nil {
		return errors.New("app not registered")
	}
	if tx == nil {
		return errors.New("nil transaction")
	}
	switch {
		case tx.Payload == nil:
			return errors.New("nil transaction payload")
		case tx.Signature == nil:
			return errors.New("nil transaction signature")
		case tx.AppId == nil:
			return errors.New("nil transaction app ID")
		case string(tx.AppId) != string(d.app.AppId):
			return errors.New("transaction app ID does not match registered app ID")
		case tx.Submitter == nil:
			return errors.New("nil transaction submitter ID")
	}
	return nil
}

func (d *dlt) Start() error {
	return errors.New("not implemented")
}

func NewDltStack(conf p2p.Config, db db.Database) (*dlt, error) {
	return &dlt {
		db: db,
	}, nil
}