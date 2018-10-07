// Copyright 2018 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"errors"
	"sync"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/p2p"
)

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
	p2p p2p.Layer
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
	return d.p2p.Start()
}

// perform handshake with the peer node
func (d *dlt) handshake(peer p2p.Peer) error {
	// Iteration 1 will not have any sync or protocol level handshake
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
			case AppConfigMsgCode:
				// deserialize the application config message from payload
				conf := AppConfig{}
				if err := msg.Decode(&conf); err != nil {
					return err
				}
				
				// application config is passed to application layer, if present, for validation
				if d.app != nil {
					if !d.peerHandler(conf) {
						// application says not a valid/compatible peer, should sharding layer filter out messages from this peer?
						// also, how to handle token attack from a malacious node faking to be application peer, but failed validation here?
						
						// TBD, handle above concerns
						return errors.New("app validation failed")
					}
				}
			case TransactionMsgCode:
				// deserialize the transaction message from payload
				tx := &Transaction{}
				if err := msg.Decode(tx); err != nil {
					return err
				}

				// transaction message should go to sharding layer
				// but, should'nt it go to endorsing layer first, to make sure its valid transaction?
				// also, should sharding layer be invoking the application callback, or should it be invoked by controller?
				// TBD, TBD, TBD ...
				
				// for iteration #1, there is only 1 message type, and always goes to application
				if d.app == nil {
					return errors.New("app not registered")
				}
				if err := d.txHandler(tx); err != nil {
					// application says not a valid transaction, peer should have validated this before sending
					// but, how can we prevent a distributed denial of service attack with this flow?
					// because, a malacious application node can join, and submit an invalid transaction that may
					// travel through the network via headless nodes, but then eventually get discarded here, in which case
					// it will trigger chain reaction of peer connection resets (if we rolled back error here)
					// also, additionally, how to preserve application account's utility token when such faulty transaction comes along?
					
					// TBD, handle above concerns
					return err
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
			// TODO: perform any cleanup here upong exit
		}()
	}
	// start listening on messages from peer node
	return d.listener(peer)
}

func NewDltStack(conf p2p.Config, db db.Database) (*dlt, error) {
	stack := &dlt {
		db: db,
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