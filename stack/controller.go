// Copyright 2018 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"errors"
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/endorsement"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/go-trust-net/common"
	"sync"
)

type DLT interface {
	// register application shard with the DLT stack
	Register(shardId []byte, name string, txHandler func(tx *dto.Transaction) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// submit a transaction to the network
	Submit(tx *dto.Transaction) error
	// start the controller
	Start() error
	// stop the controller
	Stop()
}

type dlt struct {
	app       *AppConfig
	txHandler func(tx *dto.Transaction) error
	db        repo.DltDb
	p2p       p2p.Layer
	sharder   shard.Sharder
	endorser  endorsement.Endorser
	seen      *common.Set
	lock      sync.RWMutex
}

func (d *dlt) Register(shardId []byte, name string, txHandler func(tx *dto.Transaction) error) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.app != nil {
		return errors.New("App is already registered")
	}
	d.app = &AppConfig{
		ShardId: shardId,
		Name:    name,
	}
	// app's ID need to be same as p2p node's ID
	// WHY DO WE NEED APP ID??? It will not get included in Tx Signature from submitter client
	// (since submitter may want to submit same transaction with multiple app instances) -- may be we should
	// only have shard Id in the transaction
	// ACTUALLY, this will be the Anchor (which will include app ID from DLT stack)
	d.app.AppId = d.p2p.Id()
	d.txHandler = txHandler

	// register app with sharder
	d.sharder.Register(shardId, txHandler)

//	// replay endorsement layer transactions to the registered app via sharder
//	if err := d.endorser.Replay(d.sharder.Handle); err != nil {
//		// unregister upon replay failure
//		d.unregister()
//		return err
//	}
// TBD: reply will actually happen at sharder when app registers, it already has transactions
	return nil
}

func (d *dlt) Unregister() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.unregister()
}

func (d *dlt) unregister() error {
	d.app = nil
	d.txHandler = nil
	return d.sharder.Unregister()
}

func (d *dlt) Submit(tx *dto.Transaction) error {
	if d.app == nil {
		return errors.New("app not registered")
	}
	// validate transaction
	if tx == nil {
		return errors.New("nil transaction")
	}
	if string(tx.ShardId) != string(d.app.ShardId) {
		return errors.New("incorrect shard id")
	}
	// check for TxAnchor
	// TBD

	// set NodeId for transaction
	// TBD: this will be part of TxAnchor later
	tx.NodeId = d.p2p.Id()

	switch {
	case tx.Payload == nil:
		return errors.New("nil transaction payload")
	case tx.Signature == nil:
		return errors.New("nil transaction signature")
	case tx.Submitter == nil:
		return errors.New("nil transaction submitter ID")
	}

	// send the submitted transaction to sharding layer
	// TBD

	// next in line process with endorsement layer
	// TBD: below needs to change to a different method that will check whether transaction
	// has correct TxAnchor in the submitted transaction
	if err := d.endorser.Handle(tx); err != nil {
		return err
	}

	// finally send it to p2p layer, to broadcase to others
	return d.p2p.Broadcast(tx.Signature, TransactionMsgCode, tx)
}

func (d *dlt) Start() error {
	return d.p2p.Start()
}

func (d *dlt) Stop() {
	d.p2p.Stop()
}

// perform handshake with the peer node
func (d *dlt) handshake(peer p2p.Peer) error {
	// Iteration 1 will not have any sync or protocol level handshake
	//	fmt.Printf("\nNew Peer Connection: %x\n", peer.ID())
	return nil
}

// listen on messages from the peer node
func (d *dlt) listener(peer p2p.Peer) error {
	for {
		msg, err := peer.ReadMsg()
		if err != nil {
			return err
		}
		switch msg.Code() {
		case NodeShutdownMsgCode:
			// cleanly shutdown peer connection
			d.p2p.Disconnect(peer)
			return nil

		case TransactionMsgCode:
			// deserialize the transaction message from payload
			tx := &dto.Transaction{}
			if err := msg.Decode(tx); err != nil {
				fmt.Printf("\nFailed to decode message: %s\n", err)
				return err
			}

			// validate transaction signature using transaction submitter's ID
			if !d.p2p.Verify(tx.Payload, tx.Signature, tx.Submitter) {
				return errors.New("Transaction signature invalid")
			}

			// check if message was already seen by stack
			if d.isSeen(tx.Signature) {
				continue
			}

			// transaction message should go to sharding layer
			// but, should'nt it go to endorsing layer first, to make sure its valid transaction?
			// also, should sharding layer be invoking the application callback, or should it be invoked by controller?
			// TBD, TBD, TBD ...

			// send transaction to endorsing layer for handling
			if err := d.endorser.Handle(tx); err != nil {
				// fmt.Printf("\nFailed to endorse transaction: %s\n", err)
				continue
			}

			// let sharding layer process transaction
			if err := d.sharder.Handle(tx); err != nil {
				// fmt.Printf("\nFailed to shard transaction: %s\n", err)
				continue
			}

			// mark sender of the message as seen
			peer.Seen(tx.Signature)
			d.p2p.Broadcast(tx.Signature, TransactionMsgCode, tx)

		// case 1 message type

		// case 2 message type

		// ...

		default:
			// error condition, unknown protocol message
			return errors.New("unknown protocol message recieved")
		}
	}
}

// handle a new peer node connection from p2p layer
func (d *dlt) runner(peer p2p.Peer) error {
	// initiate handshake with peer's sharding layer
	if err := d.handshake(peer); err != nil {
		return err
	} else {
		defer func() {
			// TODO: perform any cleanup here upon exit
		}()
	}
	// start listening on messages from peer node
	return d.listener(peer)
}

// mark a message as seen for stack (different from marking it seen for connected peer nodes)
func (d *dlt) isSeen(msgId []byte) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.seen.Size() > 100 {
		for i := 0; i < 20; i += 1 {
			d.seen.Pop()
		}
	}
	if !d.seen.Has(string(msgId)) {
		d.seen.Add(string(msgId))
		return false
	} else {
		return true
	}
}

func NewDltStack(conf p2p.Config, dbp db.DbProvider) (*dlt, error) {
	var db repo.DltDb
	var err error
	if db, err = repo.NewDltDb(dbp); err != nil {
		return nil, err
	} 
	stack := &dlt{
		db:   db,
		seen: common.NewSet(),
	}
	// update p2p.Config with protocol name, version and message count based on protocol specs
	conf.ProtocolName = ProtocolName
	conf.ProtocolVersion = ProtocolVersion
	conf.ProtocolLength = ProtocolLength
	if p2p, err := p2p.NewDEVp2pLayer(conf, stack.runner); err == nil {
		stack.p2p = p2p
	} else {
		return nil, err
	}
	if endorser, err := endorsement.NewEndorser(db); err == nil {
		stack.endorser = endorser
	} else {
		return nil, err
	}
	if sharder, err := shard.NewSharder(db); err == nil {
		stack.sharder = sharder
	} else {
		return nil, err
	}
	return stack, nil

}
