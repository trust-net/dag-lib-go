// Copyright 2018 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/endorsement"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"github.com/trust-net/go-trust-net/common"
	"sync"
)

type DLT interface {
	// register application shard with the DLT stack
	Register(shardId []byte, name string, txHandler func(tx dto.Transaction) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// submit a transaction to the network
	Submit(tx dto.Transaction) error
	// get a transaction Anchor for specified submitter id
	Anchor(id []byte) *dto.Anchor
	// start the controller
	Start() error
	// stop the controller
	Stop()
}

type dlt struct {
	app       *AppConfig
	txHandler func(tx dto.Transaction) error
	db        repo.DltDb
	p2p       p2p.Layer
	sharder   shard.Sharder
	endorser  endorsement.Endorser
	seen      *common.Set
	lock      sync.RWMutex
	logger    log.Logger
}

func (d *dlt) Register(shardId []byte, name string, txHandler func(tx dto.Transaction) error) error {
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

func (d *dlt) Submit(tx dto.Transaction) error {
	if d.app == nil {
		return errors.New("app not registered")
	}
	// validate transaction
	if tx == nil {
		return errors.New("nil transaction")
	}
	if string(tx.Anchor().ShardId) != string(d.app.ShardId) {
		return errors.New("incorrect shard id")
	}
	// check for TxAnchor
	// TBD

	// set NodeId for transaction
	// TBD: this will be part of TxAnchor later
	tx.Anchor().NodeId = d.p2p.Id()

	switch {
	case tx.Self().Payload == nil:
		return errors.New("nil transaction payload")
	case tx.Self().Signature == nil:
		return errors.New("nil transaction signature")
	case tx.Anchor().Submitter == nil:
		return errors.New("nil transaction submitter ID")
	}

	// check if message was already seen by stack
	if d.isSeen(tx.Id()) {
		return errors.New("seen transaction")
	}

	// send the submitted transaction for approval to sharding layer
	if err := d.sharder.Approve(tx); err != nil {
		d.logger.Debug("Submitted transaction failed to approve at sharder: %s", err)
		return err
	}

	// next in line process with endorsement layer
	// TBD: below needs to change to a different method that will check whether transaction
	// has correct TxAnchor in the submitted transaction
	if err := d.endorser.Approve(tx); err != nil {
		d.logger.Debug("Submitted transaction failed to approve at endorser: %s", err)
		return err
	}

	// finally send it to p2p layer, to broadcase to others
	return d.p2p.Broadcast(tx.Self().Id(), TransactionMsgCode, tx)
}

func (d *dlt) Anchor(id []byte) *dto.Anchor {
	a := &dto.Anchor{
		Submitter: id,
	}
	if err := d.sharder.Anchor(a); err != nil {
		d.logger.Debug("Failed to get sharder's anchor: %s", err)
		return nil
	}

	// get endorser's update on anchor
	if err := d.endorser.Anchor(a); err != nil {
		d.logger.Debug("Failed to get endorser's anchor: %s", err)
		return nil
	}

	// get p2p layer's update on anchor
	if err := d.p2p.Anchor(a); err != nil {
		d.logger.Debug("Failed to get p2p layer's anchor: %s", err)
		return nil
	}
	return a
}

func (d *dlt) Start() error {
	return d.p2p.Start()
}

func (d *dlt) Stop() {
	d.p2p.Stop()
}

// perform handshake with the peer node
func (d *dlt) handshake(peer p2p.Peer) error {
	// if there is a registered app, send shard sync message for the app's shard
	//   1) ask sharding layer for the current shard's Anchor
	//   2) ask endorsing layer for the current Anchor's update
	//   3) send the ShardSyncMsg message to peer
	a := &dto.Anchor{}
	if err := d.sharder.Anchor(a); err != nil {
		d.logger.Debug("Cannot run handshake: %s", err)
	} else if err = d.endorser.Anchor(a); err != nil {
		d.logger.Debug("Cannot run handshake: %s", err)
	} else {
		msg := NewShardSyncMsg(a)
		return peer.Send(msg.Id(), msg.Code(), msg)
	}
	return nil
}

// listen on events for a specific peer connection
func (d *dlt) peerEventsListener(peer p2p.Peer, events chan controllerEvent) {
	// fmt.Printf("Entering event listener...\n")
	done := false
	for !done {
		e := <-events
		switch e.code {
		case RECV_NewTxBlockMsg:
			tx := e.data.(dto.Transaction)

			// send transaction to endorsing layer for handling
			if err := d.endorser.Handle(tx); err != nil {
				d.logger.Error("Failed to endorse transaction: %s", err)
				continue
			}

			// let sharding layer process transaction
			if err := d.sharder.Handle(tx); err != nil {
				d.logger.Error("Failed to shard transaction: %s", err)
				continue
			}

			// mark sender of the message as seen
			id := tx.Id()
			peer.Seen(id[:])
			d.logger.Debug("Boradcasting transaction: %x", tx.Id())
			d.p2p.Broadcast(tx.Self().Id(), TransactionMsgCode, tx)

		case RECV_ShardSyncMsg:
			msg := e.data.(*ShardSyncMsg)

			// compare local anchor with remote anchor,
			// fetch anchor only for remote peer's shard,
			// since our local shard maybe different, but we may have more recent data
			// due to network updates from other nodes,
			// or if local sharder knows nothing about remot shard (sync anchor will be nil)
			myAnchor := d.sharder.SyncAnchor(msg.Anchor.ShardId)

			if myAnchor == nil || myAnchor.Weight < msg.Anchor.Weight ||
				shard.Numeric(myAnchor.ShardParent[:]) < shard.Numeric(msg.Anchor.ShardParent[:]) {
				// local shard's anchor is behind, initiate sync with remote by walking up the DAG
				req := &ShardAncestorRequestMsg{
					StartHash:    msg.Anchor.ShardParent,
					MaxAncestors: 10,
				}
				d.logger.Debug("Initiating shard sync starting by ancestors request for: %x", msg.Anchor.ShardParent)
				// save the last hash into peer's state to validate ancestors response
				peer.SetState(int(RECV_ShardAncestorResponseMsg), req.StartHash)
				// send the ancestors request to peer
				peer.Send(req.Id(), req.Code(), req)
			} else {
				d.logger.Debug("End of sync with peer: %s", peer.String())
			}

		case RECV_ShardAncestorRequestMsg:
			msg := e.data.(*ShardAncestorRequestMsg)

			// fetch the ancestors for specified shard at the starting hash
			ancestors := d.sharder.Ancestors(msg.StartHash, msg.MaxAncestors)
			req := &ShardAncestorResponseMsg{
				StartHash: msg.StartHash,
				Ancestors: ancestors,
			}
			d.logger.Debug("responding with %d ancestors from: %x", len(ancestors), msg.StartHash)
			peer.Send(req.Id(), req.Code(), req)

		case RECV_ShardAncestorResponseMsg:
			msg := e.data.(*ShardAncestorResponseMsg)

			// fetch state from peer to validate response's starting hash
			if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state != msg.StartHash {
				d.logger.Debug("start hash of ShardAncestorResponseMsg does not match saved state")
			} else {
				// walk through each ancestor to check if it's known common ancestor
				found, i := false, 0
				for ; !found && i < len(msg.Ancestors); i++ {
					// look up ancestor hash in DB
					found = d.db.GetTx(msg.Ancestors[i]) != nil
				}
				if !found && i > 0 {
					// did not find any commone ancestor, so continue walking up the DAG
					req := &ShardAncestorRequestMsg{
						StartHash:    msg.Ancestors[i-1],
						MaxAncestors: 10,
					}
					// save the last hash into peer's state to validate ancestors response
					peer.SetState(int(RECV_ShardAncestorResponseMsg), req.StartHash)
					// send the ancestors request to peer
					peer.Send(req.Id(), req.Code(), req)
				} else if found {
					d.logger.Debug("Found a known common ancestor: %x", msg.Ancestors[i-1])
					// update the RECV_ShardAncestorResponseMsg state to known common ancestor
					// to prevent any DoS attack with never ending ancestor response cycle
					peer.SetState(int(RECV_ShardAncestorResponseMsg), msg.Ancestors[i-1])

					// update the RECV_ShardChildrenResponseMsg state to validate response for children request sending next
					peer.SetState(int(RECV_ShardChildrenResponseMsg), msg.Ancestors[i-1])
					d.logger.Debug("starting DAG sync by requesting children from peer's known common ancestor")

					// now initate DAG walk through from the known common ancestor's children
					req := &ShardChildrenRequestMsg{
						Parent: msg.Ancestors[i-1],
					}
					peer.Send(req.Id(), req.Code(), req)

				}
			}

		case RECV_ShardChildrenRequestMsg:
			msg := e.data.(*ShardChildrenRequestMsg)

			// fetch the children for specified hash
			children := d.sharder.Children(msg.Parent)
			req := &ShardChildrenResponseMsg{
				Parent:   msg.Parent,
				Children: children,
			}
			d.logger.Debug("responding with %d children from: %x", len(children), msg.Parent)
			peer.Send(req.Id(), req.Code(), req)

		case RECV_ShardChildrenResponseMsg:
			msg := e.data.(*ShardChildrenResponseMsg)

			// fetch state from peer to validate response's starting hash
			if state := peer.GetState(int(RECV_ShardChildrenResponseMsg)); state != msg.Parent {
				d.logger.Debug("start hash of ShardChildrenResponseMsg does not match saved state")
			} else {
				// walk through each child to check if it's unknown, then add to child queue
				for _, child := range msg.Children {
					if d.db.GetTx(child) == nil {
						if err := peer.ShardChildrenQ().Push(child); err != nil {
							d.logger.Debug("Failed to add child to shard queue: %s", err)
							// EndOfSync
							break
						}
					}
				}
				// update the RECV_ShardChildrenResponseMsg state to null value, to prevent any repeated/cyclic DoS attack
				peer.SetState(int(RECV_ShardChildrenResponseMsg), [64]byte{})

				// emit the POP_ShardChild event for processing children queue
				events <- newControllerEvent(POP_ShardChild, nil)
			}

		case POP_ShardChild:

			// pop the first child from peer's shard children queue
			if child, err := peer.ShardChildrenQ().Pop(); err != nil {
				d.logger.Debug("Did not fetch child from shard children queue: %s", err)
				// EndOfSync
			} else {
				// send the request to fetch child transaction and its children from peer's shard DAG
				req := &TxShardChildRequestMsg{
					Hash: child.([64]byte),
				}
				// update the RECV_ShardChildrenResponseMsg state to null value, to prevent any repeated/cyclic DoS attack
				peer.SetState(int(RECV_TxShardChildResponseMsg), req.Hash)
				d.logger.Debug("Requesting transaction and shard DAG children for: %x", req.Hash)
				peer.Send(req.Id(), req.Code(), req)
			}

		case RECV_TxShardChildRequestMsg:
			msg := e.data.(*TxShardChildRequestMsg)

			// fetch the transaction for requested hash
			if tx := d.db.GetTx(msg.Hash); tx == nil {
				d.logger.Error("No transaction exists for requested hash: %x", msg.Hash)
				// this is an error condition, terminate connection with peer
				peer.Disconnect()
				// EndOfSync
			} else {
				// fetch children for this transaction from shard DAG
				children := d.sharder.Children(msg.Hash)
				req := &TxShardChildResponseMsg{
					Tx:       tx,
					Children: children,
				}
				d.logger.Debug("Sending transaction with %d children for: %x", len(children), tx.Id())
				peer.Send(req.Id(), req.Code(), req)
			}

		case SHUTDOWN:
			d.logger.Debug("Recieved SHUTDOWN event")
			done = true
			break

		default:
			d.logger.Error("Unknown event: %d", e.code)
		}
	}
	d.logger.Info("Exiting event listener...")
}

// listen on messages from the peer node
func (d *dlt) listener(peer p2p.Peer, events chan controllerEvent) error {
	for {
		msg, err := peer.ReadMsg()
		if err != nil {
			d.logger.Debug("Failed to read message: %s", err)
			return err
		}
		switch msg.Code() {
		case NodeShutdownMsgCode:
			// cleanly shutdown peer connection
			d.p2p.Disconnect(peer)
			events <- newControllerEvent(SHUTDOWN, nil)
			return nil

		case TransactionMsgCode:
			// deserialize the transaction message from payload
			tx := dto.NewTransaction(&dto.Anchor{})
			if err := msg.Decode(tx); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			}

			// validate transaction signature using transaction submitter's ID
			if !d.p2p.Verify(tx.Self().Payload, tx.Self().Signature, tx.Anchor().Submitter) {
				return errors.New("Transaction signature invalid")
			}

			// check if message was already seen by stack
			if d.isSeen(tx.Id()) {
				continue
			} else {
				// emit a RECV_NewTxBlockMsg event
				events <- newControllerEvent(RECV_NewTxBlockMsg, tx)
			}

		case ShardSyncMsgCode:
			// deserialize the shard sync message from payload
			m := &ShardSyncMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("\nFailed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardSyncMsg event
				events <- newControllerEvent(RECV_ShardSyncMsg, m)
			}

		case ShardAncestorRequestMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &ShardAncestorRequestMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardAncestorRequestMsgCode event
				events <- newControllerEvent(RECV_ShardAncestorRequestMsg, m)
			}

		case ShardAncestorResponseMsgCode:
			// deserialize the shard ancestors response message from payload
			m := &ShardAncestorResponseMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardAncestorResponseMsg event
				events <- newControllerEvent(RECV_ShardAncestorResponseMsg, m)
			}

		case ShardChildrenRequestMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &ShardChildrenRequestMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardAncestorRequestMsgCode event
				events <- newControllerEvent(RECV_ShardChildrenRequestMsg, m)
			}

		case ShardChildrenResponseMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &ShardChildrenResponseMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardAncestorRequestMsgCode event
				events <- newControllerEvent(RECV_ShardChildrenResponseMsg, m)
			}

		case TxShardChildRequestMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &TxShardChildRequestMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardAncestorRequestMsgCode event
				events <- newControllerEvent(RECV_TxShardChildRequestMsg, m)
			}

		// case 1 message type

		// case 2 message type

		// ...

		default:
			// error condition, unknown protocol message
			d.logger.Debug("Unknown protocol message recieved: %d", msg.Code())
			return errors.New("unknown protocol message recieved")
		}
	}
}

// handle a new peer node connection from p2p layer
func (d *dlt) runner(peer p2p.Peer) error {
	// initiate handshake with peer's sharding layer
	if err := d.handshake(peer); err != nil {
		d.logger.Error("Hanshake failed: %s", err)
		return err
	} else {
		defer func() {
			// TODO: perform any cleanup here upon exit
		}()
	}
	// start the event listener for this connection
	events := make(chan controllerEvent, 10)
	go d.peerEventsListener(peer, events)
	// start listening on messages from peer node
	if err := d.listener(peer, events); err != nil {
		d.logger.Info("Peer listener terminated: %s", err)
		return err
	} else {
		return nil
	}
}

// mark a message as seen for stack (different from marking it seen for connected peer nodes)
func (d *dlt) isSeen(msgId [64]byte) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.seen.Size() > 100 {
		for i := 0; i < 20; i += 1 {
			d.seen.Pop()
		}
	}
	if !d.seen.Has(msgId) {
		d.seen.Add(msgId)
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
		db:     db,
		seen:   common.NewSet(),
		logger: log.NewLogger(dlt{}),
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
