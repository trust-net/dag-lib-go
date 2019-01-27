// Copyright 2018 The trust-net Authors
// Endorsement Layer interface and implementation for DLT Statck
package endorsement

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
)

const (
	SUCCESS int = iota
	ERR_DUPLICATE
	ERR_DOUBLE_SPEND
	ERR_ORPHAN
	ERR_INVALID
)

type Endorser interface {
	// populate a transaction Anchor
	Anchor(*dto.Anchor) error
	// Handle network transaction
	Handle(tx dto.Transaction) (int, error)
	// Replace submitter history
	Replace(tx dto.Transaction) error
	// Approve submitted transaction
	Approve(tx dto.Transaction) error
	// Update submitter history for transaction
	Update(tx dto.Transaction) error
	// Provide all known shard/tx pairs for a submitter/seq
	KnownShardsTxs(submitter []byte, seq uint64) (shards [][]byte, txs [][64]byte)
}

type endorser struct {
	db repo.DltDb
}

func GenesisSubmitterTx(submitterId []byte) dto.Transaction {
	tx := dto.NewTransaction(&dto.Anchor{
		Submitter:    submitterId,
		SubmitterSeq: 0x0,
	})
	tx.Self().Signature = submitterId
	return tx
}

// validate an anchor against submitter history
func (e *endorser) isValid(a *dto.Anchor, tx dto.Transaction) (int, error) {
	// fetch submitter history for submitter's parent
	if a.SubmitterSeq > 1 {
		if parent := e.db.GetSubmitterHistory(a.Submitter, a.SubmitterSeq-1); parent == nil {
			return ERR_ORPHAN, fmt.Errorf("Unexpected submitter sequence: %d", a.SubmitterSeq)
		} else {
			// walk through known shard/tx pairs to check if parent is there
			found := false
			for _, pair := range parent.ShardTxPairs {
				if pair.TxId == a.SubmitterLastTx {
					found = true
					break
				}
			}
			if !found {
				return ERR_ORPHAN, fmt.Errorf("Unknown submitter parent: %x", a.SubmitterLastTx)
			}
		}
	}

	// ensure this is not a double spending transaction (i.e. no other transaction with same seq and shard)
	if current := e.db.GetSubmitterHistory(a.Submitter, a.SubmitterSeq); current != nil {
		// walk through known shard/tx pairs to check for double spending
		for _, pair := range current.ShardTxPairs {
			if string(pair.ShardId) == string(a.ShardId) {
				if tx == nil || tx.Id() != pair.TxId {
					return ERR_DOUBLE_SPEND, fmt.Errorf("Double spending attempt for seq: %d, shardId: %x", a.SubmitterSeq, a.ShardId)
				}
			}
		}
	}

	// anchor parameter's look good for submitter
	return SUCCESS, nil
}

// validate submitter's anchor request details
func (e *endorser) Anchor(a *dto.Anchor) error {
	// TBD: lock and unlock

	// submitter sequence should be 1 or higher
	if a == nil || a.SubmitterSeq < 1 {
		// this must be special anchor for sync
		return nil
	} else if _, err := e.isValid(a, nil); err != nil {
		return err
	} else {
		return nil
	}
}

func (e *endorser) Handle(tx dto.Transaction) (int, error) {
	// validate transaction
	// TBD
	if tx == nil || tx.Anchor() == nil || tx.Anchor().SubmitterSeq < 1 {
		return ERR_INVALID, fmt.Errorf("invalid transaction")
	}

	// check transaction against submitter history
	if res, err := e.isValid(tx.Anchor(), tx); err != nil {
		return res, err
	}

	// save the transaction
	if err := e.db.AddTx(tx); err != nil {
		return ERR_DUPLICATE, err
	}

//	// update submitter's DAG
//	if err := e.db.UpdateSubmitter(tx); err != nil {
//		return ERR_DOUBLE_SPEND, err
//	}

	// broadcast transaction
	// ^^^ this will be done by the controller if there is no error

	return SUCCESS, nil
}

func (e *endorser) Replace(tx dto.Transaction) error {
	// validate transaction
	if tx == nil || tx.Anchor() == nil || tx.Anchor().SubmitterSeq < 1 {
		return fmt.Errorf("invalid transaction")
	}

	// update submitter's history and replace if already exists
	if err := e.db.ReplaceSubmitter(tx); err != nil {
		return err
	}

	return nil
}

func (e *endorser) Approve(tx dto.Transaction) error {
	// validate transaction
	if tx == nil || tx.Anchor() == nil || tx.Anchor().SubmitterSeq < 1 {
		return fmt.Errorf("invalid transaction")
	}

	// check transaction against submitter history
	if _, err := e.isValid(tx.Anchor(), tx); err != nil {
		return err
	}

//	// update submitter's history (fails if this is double spending transaction)
//	if err := e.db.UpdateSubmitter(tx); err != nil {
//		return err
//	}

	return nil
}

func (e *endorser) Update(tx dto.Transaction) error {
//	// validate transaction
//	if tx == nil || tx.Anchor() == nil || tx.Anchor().SubmitterSeq < 1 {
//		return fmt.Errorf("invalid transaction")
//	}
//
//	// check transaction against submitter history
//	if _, err := e.isValid(tx.Anchor(), tx); err != nil {
//		return err
//	}

	// update submitter's history (fails if this is double spending transaction)
	if err := e.db.UpdateSubmitter(tx); err != nil {
		return err
	}

	return nil
}

func (e *endorser) KnownShardsTxs(submitter []byte, seq uint64) (shards [][]byte, txs [][64]byte) {
	// initialize empty lists
	shards, txs = [][]byte{}, [][64]byte{}

	// fetch submitter history for specified id/seq
	if seq > 0 {
		if history := e.db.GetSubmitterHistory(submitter, seq); history != nil {
			// walk through known shard/tx pairs and add to return list
			for _, pair := range history.ShardTxPairs {
				shards = append(shards, pair.ShardId)
				txs = append(txs, pair.TxId)
			}
		}
	}
	return
}

func NewEndorser(db repo.DltDb) (*endorser, error) {
	return &endorser{
		db: db,
	}, nil
}
