// Copyright 2018 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
//	"fmt"
	"errors"
	"sync"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/go-trust-net/common"
)

// approve if willing to accept message from application corresponding to the ID
type PeerApprover func (id []byte) bool

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
	// stop the controller
	Stop()
}

type dlt struct {
	app *AppConfig
	peerHandler PeerApprover
	txHandler NetworkTxApprover
	db db.Database
	p2p p2p.Layer
	seen *common.Set
	lock   sync.RWMutex
}

func (d *dlt) Register(app AppConfig, peerHandler PeerApprover, txHandler NetworkTxApprover) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.app != nil {
		return errors.New("App is already registered")
	}
	d.app = &app
	// app's ID need to be same as p2p node's ID
	d.app.AppId = d.p2p.Id()
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
	// update transaction's app ID
	tx.AppId = d.app.AppId

	// sign transaction using p2p layer
	if signature, err := d.p2p.Sign(tx.Payload); err != nil {
		return err
	} else {
		tx.Signature = signature
	}

	switch {
		case tx.Payload == nil:
			return errors.New("nil transaction payload")
		case tx.Signature == nil:
			return errors.New("nil transaction signature")
		case tx.Submitter == nil:
			return errors.New("nil transaction submitter ID")
	}
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
func (d *dlt) listener (peer p2p.Peer) error {
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
				tx := &Transaction{}
				if err := msg.Decode(tx); err != nil {
					return err
				}

				// validate transaction signature
				if !d.p2p.Verify(tx.Payload, tx.Signature, tx.AppId) {
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
				
				// for iteration #1, there is only 1 message type, and always goes to application
				if d.app == nil {
					return errors.New("app not registered")
				}

				// check with application if willing to accept the message
				if !d.peerHandler(tx.AppId) {
					// application rejected the application ID as peer
					// silently discard message
				} else if err := d.txHandler(tx); err != nil {
					// application says not a valid transaction, peer should have validated this before sending
					// but, how can we prevent a distributed denial of service attack with this flow?
					// because, a malacious application node can join, and submit an invalid transaction that may
					// travel through the network via headless nodes, but then eventually get discarded here, in which case
					// it will trigger chain reaction of peer connection resets (if we rolled back error here)
					// also, additionally, how to preserve application account's utility token when such faulty transaction comes along?
					
					// silently discard the transaction, and do not forward to others
					// will not deduct utility token, but then will also not propagate the message
					// so all in all its fair treatment
				} else {
					// mark sender of the message as seen
					peer.Seen(tx.Signature)
					d.p2p.Broadcast(tx.Signature, TransactionMsgCode, tx)
				}

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
func (d *dlt) runner (peer p2p.Peer) error {
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

func NewDltStack(conf p2p.Config, db db.Database) (*dlt, error) {
	stack := &dlt {
		db: db,
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
	return stack, nil
	
}