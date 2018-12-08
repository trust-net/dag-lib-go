// Copyright 2018 The trust-net Authors
// Sharding Layer interface and implementation for DLT Statck
package shard

import (
	"errors"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
)

var ShardSeqOne = [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

type Sharder interface {
	// register application shard with the DLT stack
	Register(shardId []byte, txHandler func(tx *dto.Transaction) error) error
	// unregister application shard from DLT stack
	Unregister() error
	// Handle Transaction
	Handle(tx *dto.Transaction) error
}

type sharder struct {
	db repo.DltDb

	shardId   []byte
	genesisTx *dto.Transaction
	txHandler func(tx *dto.Transaction) error
}

func GenesisShardTx(shardId []byte) *dto.Transaction {
	return &dto.Transaction{
		Payload: shardId,
		ShardId: shardId,
	}
}

func (s *sharder) Register(shardId []byte, txHandler func(tx *dto.Transaction) error) error {
	s.shardId = append(shardId)
	s.txHandler = txHandler

	// construct genesis Tx for this shard based on protocol rules
	s.genesisTx = GenesisShardTx(shardId)

	// fetch the genesis node for this shard's DAG
	if genesis := s.db.GetShardDagNode(s.genesisTx.Id()); genesis == nil {
		// unknown/new shard, save the genesis transaction
		if err := s.db.AddTx(s.genesisTx); err != nil {
			return err
		}
		// fmt.Printf("Registering genesis for shard: %x\n", shardId)
	} else {
		// fmt.Printf("Known shard Id: %x\n", shardId)
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
						if err := s.txHandler(tx); err == nil {
							// we only add children of this transaction to queue if this was a good transaction
							for _, id := range node.Children {
								// fmt.Printf("Pushing into Q: %x\n", id)
								if err := q.Push(id); err != nil {
									// had some problem
									return err
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *sharder) Unregister() error {
	s.shardId = nil
	s.txHandler = nil
	s.genesisTx = nil
	return nil
}

func (s *sharder) Handle(tx *dto.Transaction) error {
	// validate transaction
	if len(tx.ShardId) == 0 {
		return errors.New("missing shard id in transaction")
	}

	// TBD: lock and unlock

	// check for first network transactions of a new shard
	if tx.ShardSeq == ShardSeqOne {
		genesis := GenesisShardTx(tx.ShardId)
		// ensure that transaction's parent is really genesis
		if genesis.Id() != tx.ShardParent {
			return errors.New("genesis mismatch for 1st shard transaction")
		}
		// this is very first network transaction for a new shard, register the shard's genesis
		s.db.AddTx(genesis)
		// fmt.Printf("Handler genesis for shard: %x\n", genesis.ShardId)
	}

	// check if parent for the transaction is known
	if parent := s.db.GetShardDagNode(tx.ShardParent); parent == nil {
		return errors.New("parent transaction unknown for shard")
	} else {
		// add the transaction as parent's child
		// TBD
		// should we add transaction here, or should we expect that transaction has already been added by lower layer?
		// for now will assume that we'll add transaction here
		s.db.AddTx(tx)
	}

	// if an app is registered, call app's transaction handler
	if s.txHandler != nil {
		if string(s.shardId) == string(tx.ShardId) {
			return s.txHandler(tx)
		}
	}
	return nil
}

func NewSharder(db repo.DltDb) (*sharder, error) {
	return &sharder{
		db: db,
	}, nil
}
