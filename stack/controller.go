// Copyright 2018-2019 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"errors"
	"github.com/trust-net/dag-lib-go/common"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/endorsement"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"github.com/trust-net/dag-lib-go/stack/state"
	"sync"
)

type DLT interface {
	// register application shard with the DLT stack
	Register(shardId []byte, name string, txHandler func(tx dto.Transaction, state state.State) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// submit a transaction request to the network
	Submit(req *dto.TxRequest) (dto.Transaction, error)
	// get a transaction Anchor for specified submitter id
	Anchor(id []byte, seq uint64, lastTx [64]byte) *dto.Anchor
	// start the controller
	Start() error
	// stop the controller
	Stop()
	// get value for a resource from current world state for the registered shard
	GetState(key []byte) (*state.Resource, error)
}

type dlt struct {
	app       *AppConfig
	txHandler func(tx dto.Transaction, state state.State) error
	db        repo.DltDb
	p2p       p2p.Layer
	conf      *p2p.Config
	sharder   shard.Sharder
	endorser  endorsement.Endorser
	seen      *common.Set
	lock      sync.RWMutex
	logger    log.Logger
}

func (d *dlt) Register(shardId []byte, name string, txHandler func(tx dto.Transaction, state state.State) error) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.app != nil {
		d.logger.Error("Attempt to register app on already registered stack")
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
	if err := d.sharder.Register(shardId, txHandler); err != nil {
		d.logger.Error("Failed to register app with shard: %s", err)
		return err
	}

	// initiate app registration sync protocol
	if anchor, err := d.anchor(); err != nil {
		d.logger.Error("Failed to get anchor for sync: %s", err)
		return err
	} else {
		msg := NewForceShardSyncMsg(shardId, anchor)
		d.logger.Debug("Broadcasting ForceShardSync: %x", msg.Id())
		d.p2p.Broadcast(msg.Id(), msg.Code(), msg)
	}

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

func (d *dlt) validateSignatures(tx dto.Transaction) error {
	// validate transaction Anchor signature using transaction approver's ID
	if !d.p2p.Verify(tx.Anchor().Bytes(), tx.Anchor().Signature, tx.Anchor().NodeId) {
		d.logger.Debug("Invalid anchor signature for Tx: %x\n%s", tx.Id(), tx.Anchor().ToString())
		return errors.New("Anchor signature invalid")
	}

	// validate transaction request signature using transaction submitter's ID
	if !d.p2p.Verify(tx.Request().Bytes(), tx.Request().Signature, tx.Request().SubmitterId) {
		return errors.New("Payload signature invalid")
	}
	return nil
}

// protection against spam submissions
func (d *dlt) isPoW(req *dto.TxRequest) bool {
	// make sure submitter did some work
	// TBD

	return true
}

func (d *dlt) Submit(req *dto.TxRequest) (dto.Transaction, error) {
	// node needs to host a registered app for accepting transaction request
	if d.app == nil {
		return nil, errors.New("app not registered")
	}
	// validate transaction request
	switch {
	case req == nil:
		return nil, errors.New("nil transaction")
	case string(req.ShardId) != string(d.app.ShardId):
		return nil, errors.New("incorrect shard id")
	case req.Payload == nil:
		return nil, errors.New("nil transaction payload")
	case req.SubmitterId == nil:
		return nil, errors.New("nil transaction submitter ID")
	case req.Signature == nil:
		return nil, errors.New("nil transaction signature")
	case !d.isPoW(req):
		return nil, errors.New("insufficient proof of work")
	}

	// validate transaction request signature using transaction submitter's ID
	if !d.p2p.Verify(req.Bytes(), req.Signature, req.SubmitterId) {
		return nil, errors.New("Request signature invalid")
	}

	// lock shard
	if err := d.sharder.LockState(); err != nil {
		d.logger.Error("Submit: failed to get world state lock: %s", err)
		return nil, err
	}
	defer d.sharder.UnlockState()

	// build a transaction
	var tx dto.Transaction
	if a, err := d.anchor(); err != nil {
		return nil, err
	} else {
		// test my own signature
		if !d.p2p.Verify(a.Bytes(), a.Signature, a.NodeId) {
			d.logger.Debug("Invalid signature for my own anchor!!!\n%s", a.ToString())
			return nil, errors.New("Anchor signature invalid")
		}
		tx = dto.NewTransaction(req, a)
	}

	// check if message was already seen by stack
	if d.isSeen(tx.Id()) {
		d.logger.Debug("Discarding submission of seen transaction: %x", tx.Id())
		return nil, errors.New("seen transaction")
	}

	// check whether transaction has correct submitter sequencing
	if err := d.endorser.Approve(tx); err != nil {
		d.logger.Debug("Submitted transaction failed to approve at endorser: %s\ntransaction: %x", err, tx.Id())
		return nil, err
	}

	// process transaction and get approval from registered shard application instance
	if err := d.sharder.Approve(tx); err != nil {
		d.logger.Debug("Submitted transaction failed to approve at sharder: %s\ntransaction: %x", err, tx.Id())
		return nil, err
	} else {
		d.logger.Debug("Committing world state after successful transaction: %x", tx.Id())
		if err := d.endorser.Update(tx); err != nil {
			d.logger.Debug("Submitted transaction failed to update submitter history at endorser: %s\ntransaction: %x", err, tx.Id())
			return nil, err
		}

		if err := d.sharder.CommitState(tx); err != nil {
			d.logger.Debug("Submitted transaction failed to commit world state and update shard DAG: %s\ntransaction: %x", err, tx.Id())
			return nil, err
		}
	}
	// log anchor details for successfully accpeted submission
	d.logger.Debug("Submitted anchor signature for Tx: %x\n%s", tx.Id(), tx.Anchor().ToString())

	// finally send it to p2p layer, to broadcase to others
	id := tx.Id()
	if err := d.p2p.Broadcast(id[:], TransactionMsgCode, tx); err != nil {
		d.logger.Error("Submitted transaction failed to broadcast: %s", err)
	} else {
		d.logger.Debug("Submitted transaction accepted, broadcasting: %x", id)
	}
	return tx, nil
}

func (d *dlt) Anchor(id []byte, seq uint64, lastTx [64]byte) *dto.Anchor {
	// submitter sequence should be 1 or higher
	if seq < 1 {
		d.logger.Error("Incorrect submitter sequence: %d", seq)
		return nil
	}

	d.lock.Lock()
	defer d.lock.Unlock()
	if a, err := d.anchor(); err != nil {
		return nil
	} else {
		return a
	}
}

func (d *dlt) GetState(key []byte) (*state.Resource, error) {
	// fetch value from sharder
	return d.sharder.GetState(key)
}

func (d *dlt) anchor() (*dto.Anchor, error) {
	a := &dto.Anchor{}
	if err := d.sharder.Anchor(a); err != nil {
		d.logger.Debug("Failed to get sharder's anchor: %s", err)
		return nil, err
	}

	// get p2p layer's update on anchor
	if err := d.p2p.Anchor(a); err != nil {
		d.logger.Debug("Failed to get p2p layer's anchor: %s", err)
		return nil, err
	}
	return a, nil
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
	if anchor, err := d.anchor(); err != nil {
		d.logger.Debug("Cannot run handshake: %s", err)
	} else {
		msg := NewShardSyncMsg(d.app.ShardId, anchor)
		return peer.Send(msg.Id(), msg.Code(), msg)
	}
	return nil
}

func (d *dlt) handleTransaction(peer p2p.Peer, events chan controllerEvent, tx dto.Transaction, allowDupe bool) error {
	// send transaction to endorsing layer for handling
	if res, err := d.endorser.Handle(tx); err != nil {
		// check for failure reason
		switch res {
		case endorsement.ERR_DOUBLE_SPEND:
			// trigger double spending resolution
			peer.Logger().Error("Detected double spending for submitter/seq/shard: %x / %d / %x", tx.Request().SubmitterId, tx.Request().SubmitterSeq, tx.Request().ShardId)
			peer.Logger().Error("Remote peer: %s / %s", peer.Name(), peer.RemoteAddr())
			events <- newControllerEvent(ALERT_DoubleSpend, tx)
			return err
		case endorsement.ERR_DUPLICATE:
			// continue if dupe's are allowed, e.g. during sync
			if !allowDupe {
				peer.Logger().Debug("Discarding dupe transaction: %x", tx.Id())
				// we don't want to disconnect, since duplicate could happen if some other peer already forwarded the transaction
				return err
			}
		case endorsement.ERR_ORPHAN:
			// save the orphan transaction for later processing
			if err := peer.ToBeFetchedStackPush(tx); err != nil {
				peer.Logger().Debug("Failed to push into stack transaction: %x", tx.Id())
				return err
			}
			// trigger submitter sync
			req := NewSubmitterWalkUpRequestMsg(tx.Request())
			peer.Logger().Debug("Initiating submitter sync for Submitter/Seq: %x/%d\nIgnoring transaction: %x", req.Submitter, req.Seq, tx.Id())

			// not saving/checking peer state because ID for request is different response, and
			// we possibly want to run multiple sync's in parallel with same peer, so just do validation
			// on actual response message contents
			//			// save the request hash into peer's state to validate ancestors response
			//			peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), req.Id())

			// send the submitter history request to peer
			peer.Send(req.Id(), req.Code(), req)

			return err
		default:
			return err
		}
	}

	// let sharding layer process transaction
	if err := d.sharder.LockState(); err != nil {
		peer.Logger().Error("handleTransaction: failed to get world state lock: %s\nTransaction: %x", err, tx.Id())
		return err
	}
	defer d.sharder.UnlockState()
	if err := d.sharder.Handle(tx); err != nil {
		peer.Logger().Error("Failed to shard transaction: %s\nTransaction: %x", err, tx.Id())
		return err
	} else {
		peer.Logger().Debug("Commiting world state after successful transaction: %x", tx.Id())
		if err := d.endorser.Update(tx); err != nil {
			d.logger.Debug("Failed to update submitter history at endorser: %s\ntransaction: %x", err, tx.Id())
			return err
		}
		if err := d.sharder.CommitState(tx); err != nil {
			d.logger.Debug("Failed to commit world state and update shard DAG: %s\ntransaction: %x", err, tx.Id())
			return err
		}
	}

	// mark sender of the message as seen
	id := tx.Id()
	peer.Seen(id[:])
	peer.Logger().Debug("Network transaction accepted, broadcasting: %x", id)
	if err := d.p2p.Broadcast(id[:], TransactionMsgCode, tx); err != nil {
		d.logger.Error("Failed to broadcast message: %s", err)
	}
	return nil
}

func (d *dlt) toWalkUpStage(shardId []byte, shardParent [64]byte, peer p2p.Peer) error {
	// reset the seen set at peer to prepare for sync (and retransmissions)
	peer.ResetSeen()
	// make sure that sharder has genesis for unknown transaction's shard
	d.sharder.SyncAnchor(shardId)
	// build shard ancestor walk up request
	req := &ShardAncestorRequestMsg{
		StartHash:    shardParent,
		MaxAncestors: 10,
	}
	peer.Logger().Debug("Initiating shard sync for unknown transaction: %x", shardParent)
	// save the last hash into peer's state to validate ancestors response
	peer.SetState(int(RECV_ShardAncestorResponseMsg), req.StartHash)
	// send the ancestors request to peer
	return peer.Send(req.Id(), req.Code(), req)
}

// listen on events for a specific peer connection
func (d *dlt) peerEventsListener(peer p2p.Peer, events chan controllerEvent) {
	// fmt.Printf("Entering event listener...\n")
	done := false
	for !done {
		e := <-events
		switch e.code {
		case RECV_NewTxBlockMsg:
			// check if transaction's parent is known
			if tx := e.data.(dto.Transaction); d.db.GetTx(tx.Anchor().ShardParent) != nil {
				// parent is known, so process normally
				if err := d.handleTransaction(peer, events, tx, false); err != nil {
					peer.Logger().Debug("Failed to handle network transaction: %s", err)
					// TBD: should we disconnect from peer?
					// let the handler decide based on error type
					break
				}
			} else {
				// parent is unknown, so initiate sync with peer
				peer.Logger().Debug("Shard parent unknown for transaction: %x", tx.Id())
				if err := d.toWalkUpStage(tx.Request().ShardId, tx.Anchor().ShardParent, peer); err != nil {
					peer.Logger().Debug("Failed to transition to WalkUpStage: %s", err)
					peer.Disconnect()
					done = true
				}
			}

		case RECV_ShardSyncMsg:
			msg := e.data.(*ShardSyncMsg)

			// compare local anchor with remote anchor,
			// fetch anchor only for remote peer's shard,
			// since our local shard maybe different, but we may have more recent data
			// due to network updates from other nodes,
			// or if local sharder knows nothing about remot shard (sync anchor will be nil)
			myAnchor := d.sharder.SyncAnchor(msg.ShardId)

			if myAnchor == nil || msg.Anchor.Weight > myAnchor.Weight ||
				(msg.Anchor.Weight == myAnchor.Weight &&
					shard.Numeric(msg.Anchor.ShardParent[:]) > shard.Numeric(myAnchor.ShardParent[:])) {
				// local shard's anchor is behind, initiate sync with remote by walking up the DAG
				req := &ShardAncestorRequestMsg{
					StartHash:    msg.Anchor.ShardParent,
					MaxAncestors: 10,
				}
				peer.Logger().Debug("Initiating shard sync starting by ancestors request for: %x", msg.Anchor.ShardParent)
				// save the last hash into peer's state to validate ancestors response
				peer.SetState(int(RECV_ShardAncestorResponseMsg), req.StartHash)
				// send the ancestors request to peer
				peer.Send(req.Id(), req.Code(), req)
			} else {
				// explicitely set state to NOT expect any ancestor response
				peer.SetState(int(RECV_ShardAncestorResponseMsg), nil)
				peer.Logger().Debug("End of sync with peer: %s", peer.String())
			}

		case RECV_ShardAncestorRequestMsg:
			msg := e.data.(*ShardAncestorRequestMsg)

			// reset the seen set at peer to prepare for sync (and retransmissions)
			peer.ResetSeen()

			// fetch the ancestors for specified shard at the starting hash
			ancestors := d.sharder.Ancestors(msg.StartHash, msg.MaxAncestors)
			req := &ShardAncestorResponseMsg{
				StartHash: msg.StartHash,
				Ancestors: ancestors,
			}
			peer.Logger().Debug("responding with %d ancestors from: %x", len(ancestors), msg.StartHash)
			peer.Send(req.Id(), req.Code(), req)

		case RECV_ShardAncestorResponseMsg:
			msg := e.data.(*ShardAncestorResponseMsg)

			// fetch state from peer to validate response's starting hash
			if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state != msg.StartHash {
				peer.Logger().Debug("start hash of ShardAncestorResponseMsg does not match saved state")
			} else {
				// walk through each ancestor to check if it's known common ancestor
				found, i := false, 0
				for ; !found && i < len(msg.Ancestors); i++ {
					// look up ancestor hash in shard DAG
					found = d.db.GetShardDagNode(msg.Ancestors[i]) != nil
				}
				if !found && i > 0 {
					// did not find any commone ancestor, so continue walking up the DAG
					req := &ShardAncestorRequestMsg{
						StartHash:    msg.Ancestors[i-1],
						MaxAncestors: 10,
					}
					// save the last hash into peer's state to validate ancestors response
					peer.SetState(int(RECV_ShardAncestorResponseMsg), req.StartHash)
					peer.Logger().Debug("no common ancestor found, continue walk up from: %x", req.StartHash)
					// send the ancestors request to peer
					peer.Send(req.Id(), req.Code(), req)
				} else if found {
					peer.Logger().Debug("Found a known common ancestor: %x", msg.Ancestors[i-1])
					// update the RECV_ShardAncestorResponseMsg state to known common ancestor
					// to prevent any DoS attack with never ending ancestor response cycle
					peer.SetState(int(RECV_ShardAncestorResponseMsg), msg.Ancestors[i-1])

					// update the RECV_ShardChildrenResponseMsg state to validate response for children request sending next
					peer.SetState(int(RECV_ShardChildrenResponseMsg), msg.Ancestors[i-1])
					peer.Logger().Debug("starting DAG sync by requesting children from peer's known common ancestor")

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
			peer.Logger().Debug("responding with %d children from: %x", len(children), msg.Parent)
			peer.Send(req.Id(), req.Code(), req)

		case RECV_ShardChildrenResponseMsg:
			msg := e.data.(*ShardChildrenResponseMsg)

			// fetch state from peer to validate response's starting hash
			if state := peer.GetState(int(RECV_ShardChildrenResponseMsg)); state != msg.Parent {
				peer.Logger().Debug("start hash of ShardChildrenResponseMsg does not match saved state")
			} else {
				// walk through each child to check if it's unknown, then add to child queue
				for _, child := range msg.Children {
					if d.db.GetShardDagNode(child) == nil {
						if err := peer.ShardChildrenQ().Push(child); err != nil {
							peer.Logger().Debug("Failed to add child to shard queue: %s", err)
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
				peer.Logger().Debug("Did not fetch child from shard children queue: %s", err)
				// EndOfSync
			} else {
				// send the request to fetch child transaction and its children from peer's shard DAG
				req := &TxShardChildRequestMsg{
					Hash: child.([64]byte),
				}
				// update the RECV_ShardChildrenResponseMsg state to null value, to prevent any repeated/cyclic DoS attack
				peer.SetState(int(RECV_TxShardChildResponseMsg), req.Hash)
				peer.Logger().Debug("Requesting transaction and shard DAG children for: %x", req.Hash)
				if err := peer.Send(req.Id(), req.Code(), req); err != nil {
					peer.Logger().Debug("Failed to send request: %s", err)
					// pop a new child
					events <- newControllerEvent(POP_ShardChild, nil)
				}
			}

		case RECV_TxShardChildRequestMsg:
			msg := e.data.(*TxShardChildRequestMsg)

			// fetch the transaction for requested hash
			if tx := d.db.GetTx(msg.Hash); tx == nil {
				peer.Logger().Error("No transaction exists for requested hash: %x", msg.Hash)
				// this is an error condition, terminate connection with peer
				peer.Disconnect()
				// EndOfSync
			} else {
				// fetch children for this transaction from shard DAG
				children := d.sharder.Children(msg.Hash)
				req := NewTxShardChildResponseMsg(tx, children)
				if req == nil {
					d.logger.Debug("Failed to serialize transaction: %x", tx.Id())
				} else {
					peer.Logger().Debug("Sending transaction with %d children for: %x", len(children), tx.Id())
					peer.Send(req.Id(), req.Code(), req)
				}
			}

		case RECV_TxShardChildResponseMsg:
			msg := e.data.(*TxShardChildResponseMsg)

			// deserialize the transaction message from payload
			tx := dto.NewTransaction(&dto.TxRequest{}, &dto.Anchor{})
			d.logger.Debug("#####################################################")
			d.logger.Debug("// TBD: need to change dto.Transaction from interface to concrete type, so that p2p layer can provide rlp decoded transaction")
			d.logger.Debug("#####################################################")
			if err := tx.DeSerialize(msg.Bytes); err != nil {
				peer.Logger().Debug("Failed to decode message: %s", err)
				// EndOfSync
				// TBD: or should we pop next child?
				break
			}

			// fetch state from peer to validate response's starting hash
			if state := peer.GetState(int(RECV_TxShardChildResponseMsg)); state != tx.Id() {
				peer.Logger().Debug("Unexpected TxShardChildResponseMsg\nhash: %x\nExpected: %x", tx.Id(), state)
			} else {
				// validate signatures
				if err := d.validateSignatures(tx); err != nil {
					peer.Logger().Debug("TxShardChildResponseMsg transaction failed signature verification: %s", err)
					break
				}

				// mark the transaction as seen by stack
				d.isSeen(tx.Id())

				// handle transaction for each layer
				if err := d.handleTransaction(peer, events, tx, true); err == nil {
					// walk through each child to check if it's unknown, then add to child queue
					for _, child := range msg.Children {
						if err := peer.ShardChildrenQ().Push(child); err != nil {
							peer.Logger().Debug("Failed to add child to shard queue: %s", err)
							break
						}
					}
					peer.Logger().Debug("Successfully added TxShardChildResponseMsg\nhash: %x\n# of children: %x", tx.Id(), len(msg.Children))
				}

				// update the RECV_TxShardChildResponseMsg state to null value, to prevent any repeated/cyclic DoS attack
				peer.SetState(int(RECV_TxShardChildResponseMsg), [64]byte{})

				// emit the POP_ShardChild event for processing children queue
				events <- newControllerEvent(POP_ShardChild, nil)
			}

		case RECV_ForceShardSyncMsg:
			if err := d.handleRECV_ForceShardSyncMsg(peer, e.data.(*ForceShardSyncMsg)); err != nil {
				peer.Logger().Debug("Failed to handle ForceShardSyncMsg: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case RECV_SubmitterWalkUpRequestMsg:
			if err := d.handleRECV_SubmitterWalkUpRequestMsg(peer, e.data.(*SubmitterWalkUpRequestMsg)); err != nil {
				peer.Logger().Debug("Failed to handle SubmitterWalkUpRequestMsg: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case RECV_SubmitterWalkUpResponseMsg:
			if err := d.handleRECV_SubmitterWalkUpResponseMsg(peer, events, e.data.(*SubmitterWalkUpResponseMsg)); err != nil {
				peer.Logger().Debug("Failed to handle SubmitterWalkUpResponseMsg: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case RECV_SubmitterProcessDownRequestMsg:
			if err := d.handleRECV_SubmitterProcessDownRequestMsg(peer, events, e.data.(*SubmitterProcessDownRequestMsg)); err != nil {
				peer.Logger().Debug("Failed to handle SubmitterProcessDownRequestMsg: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case RECV_SubmitterProcessDownResponseMsg:
			if err := d.handleRECV_SubmitterProcessDownResponseMsg(peer, events, e.data.(*SubmitterProcessDownResponseMsg)); err != nil {
				peer.Logger().Debug("Failed to handle SubmitterProcessDownResponseMsg: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case ALERT_DoubleSpend:
			if err := d.handleALERT_DoubleSpend(peer, events, e.data.(dto.Transaction)); err != nil {
				peer.Logger().Debug("Failed to handle ALERT_DoubleSpend: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case RECV_ForceShardFlushMsg:
			if err := d.handleRECV_ForceShardFlushMsg(peer, events, e.data.(*ForceShardFlushMsg)); err != nil {
				peer.Logger().Debug("Failed to handle RECV_ForceShardFlushMsg: %s", err)
				peer.Disconnect()
				done = true
				break
			}

		case SHUTDOWN:
			peer.Logger().Debug("Recieved SHUTDOWN event")
			done = true
			break

		default:
			peer.Logger().Error("Unknown event: %d", e.code)
		}
	}
	peer.Logger().Info("Exiting event listener...")
}

func (d *dlt) handleRECV_ForceShardSyncMsg(peer p2p.Peer, msg *ForceShardSyncMsg) error {
	// reset the seen set at peer to prepare for sync (and retransmissions)
	peer.ResetSeen()
	// lock shard
	if err := d.sharder.LockState(); err != nil {
		d.logger.Error("handleRECV_ForceShardSyncMsg: failed to get world state lock: %s", err)
		return err
	}
	defer d.sharder.UnlockState()
	// compare local anchor with remote anchor,
	// fetch anchor only for remote peer's shard,
	// since our local shard maybe different, but we may have more recent data
	// due to network updates from other nodes,
	// or if local sharder knows nothing about remot shard (sync anchor will be nil)
	myAnchor := d.sharder.SyncAnchor(msg.ShardId)
	d.p2p.Anchor(myAnchor)

	if myAnchor == nil || msg.Anchor.Weight > myAnchor.Weight ||
		(msg.Anchor.Weight == myAnchor.Weight &&
			shard.Numeric(msg.Anchor.ShardParent[:]) > shard.Numeric(myAnchor.ShardParent[:])) {
		// local shard's anchor is behind, initiate sync with remote by walking up the DAG
		req := &ShardAncestorRequestMsg{
			StartHash:    msg.Anchor.ShardParent,
			MaxAncestors: 10,
		}
		peer.Logger().Debug("Initiating shard sync starting by ancestors request for: %x", msg.Anchor.ShardParent)
		// save the last hash into peer's state to validate ancestors response
		peer.SetState(int(RECV_ShardAncestorResponseMsg), req.StartHash)
		// send the ancestors request to peer
		peer.Send(req.Id(), req.Code(), req)
	} else if myAnchor != nil && (myAnchor.Weight > msg.Anchor.Weight ||
		(myAnchor.Weight == msg.Anchor.Weight && shard.Numeric(myAnchor.ShardParent[:]) > shard.Numeric(msg.Anchor.ShardParent[:]))) {
		// remote shard's anchor is behind, ask remote to initiate sync
		msg := NewShardSyncMsg(msg.ShardId, myAnchor)
		peer.Logger().Debug("Notifying peer to initiate sync: %s", peer.String())
		peer.Send(msg.Id(), msg.Code(), msg)
	} else {
		peer.Logger().Debug("Shard in sync with peer: %s", peer.String())
	}
	return nil
}

func (d *dlt) handleRECV_SubmitterWalkUpRequestMsg(peer p2p.Peer, msg *SubmitterWalkUpRequestMsg) error {
	// reset the seen set at peer to prepare for sync (and retransmissions)
	peer.ResetSeen()
	// fetch the submitter/seq's history for known shard/transaction pairs
	req := NewSubmitterWalkUpResponseMsg(msg)
	req.Shards, req.Transactions = d.endorser.KnownShardsTxs(msg.Submitter, msg.Seq)
	peer.Logger().Debug("responding with %d pairs for: %x / %d", len(req.Shards), req.Submitter, req.Seq)
	peer.Send(req.Id(), req.Code(), req)
	return nil
}

func (d *dlt) handleRECV_SubmitterWalkUpResponseMsg(peer p2p.Peer, events chan controllerEvent, msg *SubmitterWalkUpResponseMsg) error {
	// confirm that we did receive pairs for non-zero sequence
	if msg.Seq > 0 && len(msg.Shards) < 1 {
		peer.Logger().Error("Recieved zero pairs for submmiter/seq: %x [%d]", msg.Submitter, msg.Seq)
		return errors.New("zero pairs non-zero seq")
	}
	// confirm that number of shards and transactions is same
	if len(msg.Shards) != len(msg.Transactions) {
		peer.Logger().Error("Recieved %d shards but %d transactions for submmiter/seq: %x [%d]", len(msg.Shards), len(msg.Transactions), msg.Submitter, msg.Seq)
		return errors.New("shard transaction pair count mismatch")
	}
	// not saving/checking peer state because ID for request is different response, and
	// we possibly want to run multiple sync's in parallel with same peer, so just do validation
	// on actual response message contents
	//	// confirm that message id matches expected peer state
	//	if data := peer.GetState(int(RECV_SubmitterWalkUpResponseMsg)); data == nil || string(data.([]byte)) != string(msg.Id()) {
	//		d.logger.Error("Recieved unexpected SubmitterWalkUpResponseMsg for submmiter/seq: %x / %d", msg.Submitter, msg.Seq)
	//		return errors.New("unexpected SubmitterWalkUpResponseMsg")
	//	} else {
	//		// reset the peer state
	//		peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), nil)
	//	}

	// fetch the local submitter/seq's history for known shard/transaction pairs
	shards, transactions := d.endorser.KnownShardsTxs(msg.Submitter, msg.Seq)
	shardMap := make(map[string]int)
	for i, shard := range shards {
		shardMap[string(shard)] = i
	}
	// compare recieved pairs with known local pairs
	allMatched := true
	for i, shard := range msg.Shards {
		if indx, found := shardMap[string(shard)]; !found {
			// not doing shard sync, just continueing walk up till we find seq which is same as local
			// and will do any additional shard sync when actually processing transaction while walking down
			//			// shard unknown, initiate sync
			//			a := &dto.Anchor{
			//				ShardId:     shard,
			//				ShardParent: msg.Transactions[i],
			//			}
			//			if err := d.toWalkUpStage(a, peer); err != nil {
			//				return err
			//			}

			// this shard did not match
			allMatched = false

		} else if transactions[indx] != msg.Transactions[i] {
			// this shard's transactions did not match
			allMatched = false
			// will not do error handling here, will just keep walking up in submitter history till we reach
			// a sequence that matches locally -- or till we reach to seq 1, and then process transactions during
			// down walk, that's when will handle alerts/errors etc, so there is only one place to take care
			// double spend, i.e., transaction mismatch for matched shard
			//			d.logger.Error("Detected double spending for submitter/seq/shard: %x / %d / %x\nLocal Tx: %x\nRemote Tx: %x", msg.Submitter, msg.Seq, shard, transactions[indx], msg.Transactions[i])
			//			events <- newControllerEvent(ALERT_DoubleSpend, shard)
			//			return nil
		}

	}
	// if all pairs matched then transition to ProcessDownStage for submitter sync
	if allMatched {
		// build a walk down request
		req := NewSubmitterProcessDownRequestMsg(msg)
		//		// save hash for validation of response
		//		peer.SetState(int(RECV_SubmitterProcessDownResponseMsg), req.Id())
		// send message to peer
		peer.Logger().Debug("requesting submitter history for: %x / %d", req.Submitter, req.Seq)
		peer.Send(req.Id(), req.Code(), req)
	} else {
		// continue walking up
		req := &SubmitterWalkUpRequestMsg{
			Submitter: msg.Submitter,
			Seq:       msg.Seq - 1,
		}
		peer.Logger().Debug("continuing walk up for Submitter/Seq: %x/%d", req.Submitter, req.Seq)
		//		// save the request hash into peer's state to validate ancestors response
		//		peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), req.Id())
		// send the submitter history request to peer
		peer.Send(req.Id(), req.Code(), req)
	}
	return nil
}

func (d *dlt) handleRECV_SubmitterProcessDownRequestMsg(peer p2p.Peer, events chan controllerEvent, msg *SubmitterProcessDownRequestMsg) error {
	// fetch the submitter/seq's history for known shard/transaction pairs
	_, txIds := d.endorser.KnownShardsTxs(msg.Submitter, msg.Seq)
	txs := []dto.Transaction{}
	// fetch actual transactions that are in history
	for _, id := range txIds {
		txs = append(txs, d.db.GetTx(id))
	}
	// build the response
	if resp := NewSubmitterProcessDownResponseMsg(msg, txs); resp != nil {
		peer.Logger().Debug("responding with %d transactions for: %x / %d", len(resp.TxBytes), resp.Submitter, resp.Seq)
		peer.Send(resp.Id(), resp.Code(), resp)
	} else {
		return errors.New("Failed to create a SubmitterProcessDownResponseMsg")
	}
	return nil
}

func (d *dlt) handleRECV_SubmitterProcessDownResponseMsg(peer p2p.Peer, events chan controllerEvent, msg *SubmitterProcessDownResponseMsg) error {
	// confirm that we did receive response for non-zero sequence
	if msg.Seq < 1 {
		peer.Logger().Error("Recieved incorrect submmiter/seq: %x [%d]", msg.Submitter, msg.Seq)
		return errors.New("zero seq")
	}
	// if response does not have any transactions then EndOfSync
	if len(msg.TxBytes) == 0 {
		peer.Logger().Debug("End of sync at submmiter/seq: %x [%d]", msg.Submitter, msg.Seq)
		return nil
	}
	// process each transaction in the response
	for _, bytes := range msg.TxBytes {
		// deserialize the transaction message from bytes
		d.logger.Debug("#####################################################")
		d.logger.Debug("// TBD: need to change dto.Transaction from interface to concrete type, so that p2p layer can provide rlp decoded transaction")
		d.logger.Debug("#####################################################")
		tx := dto.NewTransaction(&dto.TxRequest{}, &dto.Anchor{})
		if err := tx.DeSerialize(bytes); err != nil {
			peer.Logger().Error("Failed to decode transaction from SubmitterProcessDownResponseMsg: %s", err)
			return err
		}

		// validate transaction signatures using transaction submitter's ID
		if err := d.validateSignatures(tx); err != nil {
			peer.Logger().Error("SubmitterProcessDownResponseMsg failed signature validation: %s", err)
			return err
		}

		// if transaction's submitter/seq does not match response's submitter/seq -- disconnect peer and abort
		if string(tx.Request().SubmitterId) != string(msg.Submitter) || tx.Request().SubmitterSeq != msg.Seq {
			peer.Logger().Error("Included transaction: %x / %d does not match SubmitterProcessDownResponseMsg: %x / %d", tx.Request().SubmitterId, tx.Request().SubmitterSeq, msg.Submitter, msg.Seq)
			return errors.New("transaction mismatch")
		}

		// mark the transaction as seen by stack
		d.isSeen(tx.Id())

		// check if need to initiate a shard sync with peer due to this transaction
		// check if transaction's parent is known
		if d.db.GetTx(tx.Anchor().ShardParent) != nil {
			// parent is known, so process normally
			if err := d.handleTransaction(peer, events, tx, false); err != nil {
				peer.Logger().Debug("Failed to handle SubmitterProcessDownResponseMsg transaction: %s", err)
				// we got an error condition (i.e. either got double spend, or an orphan transaction)
				// terminate processing and return, but do not disconnect with peer
				return nil
			}
		} else {
			// parent is unknown, so initiate sync with peer
			if err := d.toWalkUpStage(tx.Request().ShardId, tx.Anchor().ShardParent, peer); err != nil {
				peer.Logger().Debug("Failed to transition to WalkUpStage: %s", err)
				return errors.New("transition to WalkUpStage failed")
			} else {
				// initiated shard sync, so stop processing for now
				return nil
			}
		}
	}
	// send a SubmitterProcessDownRequestMsg with next higher sequence to peer
	req := &SubmitterProcessDownRequestMsg{
		Submitter: msg.Submitter,
		Seq:       msg.Seq + 1,
	}
	peer.Logger().Debug("continuing processing down for Submitter/Seq: %x/%d", req.Submitter, req.Seq)
	// send the submitter history request to peer
	return peer.Send(req.Id(), req.Code(), req)
}

func (d *dlt) handleALERT_DoubleSpend(peer p2p.Peer, events chan controllerEvent, remoteTx dto.Transaction) error {
	// reset the seen set at peer to prepare for sync (and retransmissions)
	peer.ResetSeen()
	// lock sharder
	if err := d.sharder.LockState(); err != nil {
		d.logger.Error("handleALERT_DoubleSpend: failed to get world state lock: %s", err)
		return err
	}
	defer d.sharder.UnlockState()

	// fetch local transaction for same submitter/seq/shard
	shards, transactions := d.endorser.KnownShardsTxs(remoteTx.Request().SubmitterId, remoteTx.Request().SubmitterSeq)
	shardMap := make(map[string]int)
	for i, shard := range shards {
		shardMap[string(shard)] = i
	}
	var localTx dto.Transaction
	if indx, found := shardMap[string(remoteTx.Request().ShardId)]; !found {
		peer.Logger().Error("did not find local shard for submitter/seq: %x / %d", remoteTx.Request().SubmitterId, remoteTx.Request().SubmitterSeq)
		return nil
	} else if localTx = d.db.GetTx(transactions[indx]); localTx == nil {
		peer.Logger().Error("did not find my own local transaction: %x", transactions[indx])
		// local corruption, abort everything
		return errors.New("local DB corruption")
	}
	peer.Logger().Error("Local Double Spending Tx: %x\nRemote Double Spending Tx: %x", localTx.Id(), remoteTx.Id())
	// compare local with remote
	// first compare weights, if equal then compare numeric hash
	if localId, remoteId := localTx.Id(), remoteTx.Id(); localTx.Anchor().Weight > remoteTx.Anchor().Weight ||
		(localTx.Anchor().Weight == remoteTx.Anchor().Weight &&
			shard.Numeric(localId[:]) > shard.Numeric(remoteId[:])) {
		// we should replace the local submitter history to use the winning transaction
		// so that don't get into loop when sync and remote sends the winning transaction
		// but local history still has old transaction
		if err := d.endorser.Replace(remoteTx); err != nil {
			peer.Logger().Error("Failed to update local submitter history: %s", err)
			return err
		}
		if err := d.sharder.Flush(remoteTx.Request().ShardId); err != nil {
			return err
		} else {
			peer.Logger().Debug("flushed local shard")
			// initiate a force shard sync for the flushed shard with peer
			// we need to force the shard sync because if peer is headless
			// then regular handshake will not result in sync
			myAnchor := d.sharder.SyncAnchor(remoteTx.Request().ShardId)
			d.p2p.Anchor(myAnchor)
			msg := NewForceShardSyncMsg(remoteTx.Request().ShardId, myAnchor)
			peer.Logger().Debug("sending ForceShardSync: %x", msg.Id())
			peer.Send(msg.Id(), msg.Code(), msg)
		}
	} else {
		// send peer alert to flush
		msg := NewForceShardFlushMsg(localTx)
		peer.Logger().Debug("Alerting remote peer to flush and re-sync")
		peer.Send(msg.Id(), msg.Code(), msg)
	}
	return nil
}

func (d *dlt) handleRECV_ForceShardFlushMsg(peer p2p.Peer, events chan controllerEvent, msg *ForceShardFlushMsg) error {
	// lock sharder
	if err := d.sharder.LockState(); err != nil {
		d.logger.Error("handleRECV_ForceShardFlushMsg: failed to get world state lock: %s", err)
		return err
	}
	defer d.sharder.UnlockState()

	// deserialize remote transaction from message
	d.logger.Debug("#####################################################")
	d.logger.Debug("// TBD: need to change dto.Transaction from interface to concrete type, so that p2p layer can provide rlp decoded transaction")
	d.logger.Debug("#####################################################")
	remoteTx := dto.NewTransaction(&dto.TxRequest{}, &dto.Anchor{})
	if err := remoteTx.DeSerialize(msg.Bytes); err != nil {
		peer.Logger().Debug("Failed to decode remote message: %s", err)
		return err
	}

	// fetch local transaction for same submitter/seq/shard
	shards, transactions := d.endorser.KnownShardsTxs(remoteTx.Request().SubmitterId, remoteTx.Request().SubmitterSeq)
	shardMap := make(map[string]int)
	for i, shard := range shards {
		shardMap[string(shard)] = i
	}
	var localTx dto.Transaction
	if indx, found := shardMap[string(remoteTx.Request().ShardId)]; !found {
		peer.Logger().Error("did not find local shard for submitter/seq: %x / %d", remoteTx.Request().SubmitterId, remoteTx.Request().SubmitterSeq)
		// we received incorrect request, disconnect
		return errors.New("incorred request to flush shard")
	} else if localTx = d.db.GetTx(transactions[indx]); localTx == nil {
		peer.Logger().Error("did not find my own local transaction: %x", transactions[indx])
		// local corruption, abort everything
		return errors.New("local DB corruption")
	}
	// compare local with remote
	// first compare weights, if equal then compare numeric hash
	if localId, remoteId := localTx.Id(), remoteTx.Id(); localTx.Anchor().Weight > remoteTx.Anchor().Weight ||
		(localTx.Anchor().Weight == remoteTx.Anchor().Weight &&
			shard.Numeric(localId[:]) > shard.Numeric(remoteId[:])) {
		if err := d.sharder.Flush(remoteTx.Request().ShardId); err != nil {
			return err
		} else {
			// reset the seen set at peer to prepare for sync (and retransmissions)
			peer.ResetSeen()
			peer.Logger().Debug("flushed local shard and reset seen set")
			// initiate a force shard sync for the flushed shard with peer
			// we need to force the shard sync because if peer is headless
			// then regular handshake will not result in sync
			myAnchor := d.sharder.SyncAnchor(remoteTx.Request().ShardId)
			d.p2p.Anchor(myAnchor)
			msg := NewForceShardSyncMsg(remoteTx.Request().ShardId, myAnchor)
			peer.Logger().Debug("sending ForceShardSync: %x", msg.Id())
			peer.Send(msg.Id(), msg.Code(), msg)
		}
	} else {
		// we received incorrect request, disconnect
		return errors.New("incorred request to flush shard")
	}
	return nil
}

// listen on messages from the peer node
func (d *dlt) listener(peer p2p.Peer, events chan controllerEvent) error {
	for {
		msg, err := peer.ReadMsg()
		if err != nil {
			peer.Logger().Debug("Failed to read message: %s", err)
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
			tx := dto.NewTransaction(&dto.TxRequest{}, &dto.Anchor{})
			if err := msg.Decode(tx); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			}

			// validate signatures
			if err := d.validateSignatures(tx); err != nil {
				peer.Logger().Debug("Network transaction failed signature verification: %s", err)
				return err
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
				// emit a RECV_ShardAncestorRequestMsg event
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
				// emit a RECV_ShardChildrenRequestMsg event
				events <- newControllerEvent(RECV_ShardChildrenRequestMsg, m)
			}

		case ShardChildrenResponseMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &ShardChildrenResponseMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ShardChildrenResponseMsg event
				events <- newControllerEvent(RECV_ShardChildrenResponseMsg, m)
			}

		case TxShardChildRequestMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &TxShardChildRequestMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_TxShardChildRequestMsg event
				events <- newControllerEvent(RECV_TxShardChildRequestMsg, m)
			}

		case TxShardChildResponseMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &TxShardChildResponseMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_TxShardChildResponseMsg event
				events <- newControllerEvent(RECV_TxShardChildResponseMsg, m)
			}

		case ForceShardSyncMsgCode:
			// deserialize the shard ancestors request message from payload
			m := &ForceShardSyncMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ForceShardSyncMsg event
				events <- newControllerEvent(RECV_ForceShardSyncMsg, m)
			}

		case SubmitterWalkUpRequestMsgCode:
			// deserialize the submitter history request message from payload
			m := &SubmitterWalkUpRequestMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_SubmitterWalkUpRequestMsg event
				events <- newControllerEvent(RECV_SubmitterWalkUpRequestMsg, m)
			}

		case SubmitterWalkUpResponseMsgCode:
			// deserialize the submitter history request message from payload
			m := &SubmitterWalkUpResponseMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_SubmitterWalkUpResponseMsg event
				events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, m)
			}

		case SubmitterProcessDownRequestMsgCode:
			// deserialize the submitter history request message from payload
			m := &SubmitterProcessDownRequestMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_SubmitterProcessDownRequestMsg event
				events <- newControllerEvent(RECV_SubmitterProcessDownRequestMsg, m)
			}

		case SubmitterProcessDownResponseMsgCode:
			// deserialize the submitter history request message from payload
			m := &SubmitterProcessDownResponseMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_SubmitterProcessDownResponseMsg event
				events <- newControllerEvent(RECV_SubmitterProcessDownResponseMsg, m)
			}

		case ForceShardFlushMsgCode:
			// deserialize the submitter history request message from payload
			m := &ForceShardFlushMsg{}
			if err := msg.Decode(m); err != nil {
				d.logger.Debug("Failed to decode message: %s", err)
				return err
			} else {
				// emit a RECV_ForceShardFlushMsg event
				events <- newControllerEvent(RECV_ForceShardFlushMsg, m)
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
	localAddr, remoteAddr := d.conf.ListenAddr, "unknown"
	if peer.LocalAddr() != nil {
		localAddr = peer.LocalAddr().String()
	}
	if peer.RemoteAddr() != nil {
		remoteAddr = peer.RemoteAddr().String()
	}
	peer.SetLogger(log.NewLogger(d.conf.Name + "/" + localAddr + " | " + peer.Name() + "/" + remoteAddr))
	peer.Logger().Info("Connected with remote node: %s [%s]", peer.Name(), peer.String())

	// initiate handshake with peer's sharding layer
	if err := d.handshake(peer); err != nil {
		peer.Logger().Error("Hanshake failed: %s", err)
		return err
	} else {
		defer func() {
			peer.Logger().Info("Disconnecting with remote node: %s", peer.Name())
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
		logger: log.NewLogger(conf.Name),
		conf:   &conf,
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
	if sharder, err := shard.NewSharder(db, dbp); err == nil {
		stack.sharder = sharder
	} else {
		return nil, err
	}
	return stack, nil

}
