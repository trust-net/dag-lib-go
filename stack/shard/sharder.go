// Copyright 2018 The trust-net Authors
// Sharding Layer interface and implementation for DLT Statck
package shard

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/state"
	"sync"
)

var ShardSeqOne = uint64(0x01)

type Sharder interface {
	// register application shard with the DLT stack
	Register(shardId []byte, txHandler func(tx dto.Transaction, state state.State) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// populate a transaction Anchor
	Anchor(a *dto.Anchor) error
	// provide anchor for syncing with specified shard
	SyncAnchor(shardId []byte) *dto.Anchor
	// provide max ancestors from specified start hash
	Ancestors(startHash [64]byte, max uint64) [][64]byte
	// provide children of specified hash
	Children(parent [64]byte) [][64]byte
	// Approve submitted transaction
	Approve(tx dto.Transaction) error
	// Handle Transaction
	Handle(tx dto.Transaction) error
	// get value for a resource from current world state for the registered shard
	GetState(key []byte) (*state.Resource, error)
}

type sharder struct {
	db  repo.DltDb
	dbp db.DbProvider

	shardId       []byte
	genesisTx     dto.Transaction
	txHandler     func(tx dto.Transaction, state state.State) error
	worldState    state.State
	useWorldState sync.RWMutex
}

func GenesisShardTx(shardId []byte) dto.Transaction {
	tx := dto.NewTransaction(&dto.Anchor{
		ShardId: shardId,
	})
	tx.Self().Signature = shardId
	return tx
}

func (s *sharder) Register(shardId []byte, txHandler func(tx dto.Transaction, state state.State) error) error {
	s.shardId = append(shardId)
	s.txHandler = txHandler
	if state, err := state.NewWorldState(s.dbp, shardId); err == nil {
		s.worldState = state
	} else {
		return fmt.Errorf("Failed to get world state reference: %s", err)
	}
	// lock world state for replay
	s.useWorldState.Lock()
	defer s.useWorldState.Unlock()

	// construct genesis Tx for this shard based on protocol rules
	s.genesisTx = GenesisShardTx(shardId)

	// fetch the genesis node for this shard's DAG
	var genesis *repo.DagNode
	if genesis = s.db.GetShardDagNode(s.genesisTx.Id()); genesis == nil {
		// unknown/new shard, save the genesis transaction
		if err := s.db.AddTx(s.genesisTx); err != nil {
			return err
		} else if err = s.db.UpdateShard(s.genesisTx); err != nil {
			return err
		}
		// now retry to fetch genesis node
		if genesis = s.db.GetShardDagNode(s.genesisTx.Id()); genesis == nil {
			// still can't get it, abort
			return fmt.Errorf("Cannot fetch genesis DAG node")
		}

		// fmt.Printf("Registering genesis for shard: %x\n", shardId)
	}
	// known shard, so replay transactions to the registered app
	// by performing a breadth first tranversal on shard's DAG and calling
	// app's transaction handler
	q, _ := repo.NewQueue(100)
	// add genesis's children's node ids to the queue
	for _, id := range genesis.Children {
		// fmt.Printf("Pushing into Q: %x\n", id)
		q.Push(id)
	}
	for q.Count() > 0 {
		// pop a node id from traversal queue
		if value, err := q.Pop(); err != nil {
			// had some problem
			return err
		} else {
			// get nodeId from popped interface
			id, _ := value.([64]byte)
			// fmt.Printf("GetShardDagNode: %x\n", value)
			// fetch shard DAG node from DB for this id
			if node := s.db.GetShardDagNode(id); node != nil {
				// fetch transaction for this node
				if tx := s.db.GetTx(node.TxId); tx != nil {
					// fmt.Printf("GetTx: %x\n", tx.Id())
					// replay transaction to the app
					if err := s.txHandler(tx, s.worldState); err == nil {
						// we only add children of this transaction to queue if this was a good transaction
						for _, id := range node.Children {
							// fmt.Printf("Pushing into Q: %x\n", id)
							if err := q.Push(id); err != nil {
								// had some problem
								s.Unregister()
								return err
							}
						}
					} else {
						s.Unregister()
						return err
					}
				}
			}
		}
	}
	// transaction replay successful, persist world state
	if err := s.worldState.Persist(); err != nil {
		return err
	}
	return nil
}

func (s *sharder) Unregister() error {
	s.shardId = nil
	s.txHandler = nil
	s.genesisTx = nil
	///////////////////////////////////////////////////////
	// TBD: remove below when we start persisting last processed transaction during unregister,
	// and do not replay previosuly processed transactions during register
	s.worldState.Reset()
	///////////////////////////////////////////////////////
	s.worldState = nil
	return nil
}

func Numeric(id []byte) uint64 {
	num := uint64(0)
	for _, b := range id {
		num += uint64(b)
	}
	return num
}

func (s *sharder) Anchor(a *dto.Anchor) error {
	// TBD: lock and unlock

	// make sure app is registered
	if s.shardId == nil {
		return fmt.Errorf("app not registered")
	} else {
		return s.updateAnchor(s.shardId, a)
	}
}

func (s *sharder) SyncAnchor(shardId []byte) *dto.Anchor {
	a := &dto.Anchor{}
	if err := s.updateAnchor(shardId, a); err != nil {
		return nil
	}
	return a
}

func (s *sharder) updateAnchor(shardId []byte, a *dto.Anchor) error {

	// assign shard ID of specified shard
	a.ShardId = shardId

	// get tips of the shard's DAG
	tips := s.db.ShardTips(shardId)

	if len(tips) == 0 {
		// create the genesis transaction for this unknown shard
		genesis := GenesisShardTx(shardId)
		if err := s.db.AddTx(genesis); err != nil {
			// ignore, there is already a genesis transaction in DB
		} else if err = s.db.UpdateShard(genesis); err != nil {
			return err
		}
		return fmt.Errorf("shard unknown")
	}

	// find the deepest node as parent
	parent := s.db.GetShardDagNode(tips[0])
	uncles := [][64]byte{}
	weight := parent.Depth
	for i := 1; i < len(tips); i += 1 {
		node := s.db.GetShardDagNode(tips[i])
		weight += node.Depth
		if parent.Depth < node.Depth {
			uncles = append(uncles, parent.TxId)
			parent = node
		} else if parent.Depth == node.Depth && Numeric(parent.TxId[:]) < Numeric(node.TxId[:]) {
			uncles = append(uncles, parent.TxId)
			parent = node
		} else {
			uncles = append(uncles, node.TxId)
		}
	}

	// assign shard DAG's parent node ID to anchor
	a.ShardParent = parent.TxId

	// assign sequence 1 greater than DAG's parent node
	a.ShardSeq = parent.Depth + 1

	// assign weight as summation of all tip's depth + 1
	a.Weight = weight + 1

	// assign uncles to anchor
	a.ShardUncles = uncles
	return nil
}

func (s *sharder) Ancestors(startHash [64]byte, max uint64) [][64]byte {
	ancestors := make([][64]byte, 0, max)
	for current := s.db.GetShardDagNode(startHash); max > 0 && current != nil; {
		if current = s.db.GetShardDagNode(current.Parent); current != nil {
			ancestors = append(ancestors, current.TxId)
			max -= 1
		}
	}
	return ancestors
}

func (s *sharder) Children(parent [64]byte) [][64]byte {
	for parentNode := s.db.GetShardDagNode(parent); parentNode != nil; {
		return parentNode.Children
	}
	return nil
}

func (s *sharder) Approve(tx dto.Transaction) error {
	// make sure app is registered
	if s.shardId == nil {
		return fmt.Errorf("app not registered")
	}

	// validate transaction
	if len(tx.Anchor().ShardId) == 0 {
		return fmt.Errorf("missing shard id in transaction")
	} else if string(s.shardId) != string(tx.Anchor().ShardId) {
		return fmt.Errorf("incorrect shard Id")
	}

	// check if parent for the transaction is known
	if parent := s.db.GetShardDagNode(tx.Anchor().ShardParent); parent == nil {
		return fmt.Errorf("parent transaction unknown for shard")
	} else {
		// should we add transaction here, or should we expect that transaction will be added by lower layer?
		// for submissions, we'll add transaction here
		if err := s.db.AddTx(tx); err != nil {
			return err
		}
		// update the shard's DAG and Tips
		if err := s.db.UpdateShard(tx); err != nil {
			return err
		}

		// process transaction via application's callback
		// lock the world state so that no other transaction can process in parallel
		s.useWorldState.Lock()
		defer s.useWorldState.Unlock()
		if err := s.txHandler(tx, s.worldState); err != nil {
			return err
		}

		// application processed transaction successfully, persist
		if err := s.worldState.Persist(); err != nil {
			return err
		}
	}
	return nil
}

func (s *sharder) Handle(tx dto.Transaction) error {
	// validate transaction
	if len(tx.Anchor().ShardId) == 0 {
		return fmt.Errorf("missing shard id in transaction")
	}

	// TBD: lock and unlock

	// check for first network transactions of a new shard
	if tx.Anchor().ShardSeq == ShardSeqOne {
		genesis := GenesisShardTx(tx.Anchor().ShardId)
		// ensure that transaction's parent is really genesis
		if genesis.Id() != tx.Anchor().ShardParent {
			return fmt.Errorf("genesis mismatch for 1st shard transaction")
		}
		// this is very first network transaction for a new shard, register the shard's genesis
		if err := s.db.AddTx(genesis); err != nil {
			// ignore, there is already a genesis transaction in DB
		} else if err = s.db.UpdateShard(genesis); err != nil {
			return err
		}
		// fmt.Printf("Handler genesis for shard: %x\n", genesis.ShardId)
	}

	// check if parent for the transaction is known
	if parent := s.db.GetShardDagNode(tx.Anchor().ShardParent); parent == nil {
		return fmt.Errorf("parent transaction unknown for shard")
	} else {
		// should we add transaction here, or should we expect that transaction has already been added by lower layer?
		// for network transactions we'll assume that it has already been added by endorsement layer

		// update shard's DAG and Tips in DB
		if err := s.db.UpdateShard(tx); err != nil {
			return err
		}
	}

	// if an app is registered, call app's transaction handler
	if s.txHandler != nil && string(s.shardId) == string(tx.Anchor().ShardId) {
		// lock the world state so that no other transaction can process in parallel
		s.useWorldState.Lock()
		defer s.useWorldState.Unlock()
		if err := s.txHandler(tx, s.worldState); err != nil {
			return err
		}

		// application processed transaction successfully, persist
		if err := s.worldState.Persist(); err != nil {
			return err
		}
	}
	return nil
}

func (s *sharder) GetState(key []byte) (*state.Resource, error) {
	// make sure app is registered
	if s.shardId == nil {
		return nil, fmt.Errorf("app not registered")
	} else {
		// fetch resource from world state
		if r, err := s.worldState.Get(key); err != nil {
			return nil, err
		} else {
			return r, nil
		}
	}
}

func NewSharder(db repo.DltDb, dbp db.DbProvider) (*sharder, error) {
	return &sharder{
		db:  db,
		dbp: dbp,
	}, nil
}
