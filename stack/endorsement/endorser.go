// Copyright 2018 The trust-net Authors
// Endorsement Layer interface and implementation for DLT Statck
package endorsement

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
)

type Endorser interface {
	// populate a transaction Anchor
	Anchor(*dto.Anchor) error
	// Handle network transaction
	Handle(tx dto.Transaction) error
	// Approve submitted transaction
	Approve(tx dto.Transaction) error
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

// validate submitter's anchor request details
func (e *endorser) Anchor(a *dto.Anchor) error {
	// TBD: lock and unlock

	// submitter sequence should be 1 or higher
	if a == nil || a.SubmitterSeq < 1 {
		// this must be special anchor for sync
		return nil
	}

	// fetch submitter history for submitter's parent
	if a.SubmitterSeq > 1 {
		if parent := e.db.GetSubmitterHistory(a.Submitter, a.SubmitterSeq-1); parent == nil {
			return fmt.Errorf("Unexpected submitter sequence: %d", a.SubmitterSeq)
		} else if parent.TxId != a.SubmitterLastTx {
			return fmt.Errorf("Incorrect submitter parent: %x", a.SubmitterLastTx)
		}
	}

	// ensure this is not a double spending transaction (i.e. no other transaction with same seq)
	if node := e.db.GetSubmitterHistory(a.Submitter, a.SubmitterSeq); node != nil {
		return fmt.Errorf("Double spending transaction: %x", node.TxId)
	}

	// anchor parameter's look good for submitter
	return nil
}

func (e *endorser) Handle(tx dto.Transaction) error {
	// validate transaction
	// TBD
	if tx == nil {
		return fmt.Errorf("invalid transaction")
	}

	if err := e.db.AddTx(tx); err != nil {
		return err
	}

	// update submitter's DAG
	if err := e.db.UpdateSubmitter(tx); err != nil {
		return err
	}

	// broadcast transaction
	// ^^^ this will be done by the controller if there is no error

	return nil
}

func (e *endorser) Approve(tx dto.Transaction) error {
	// validate transaction
	if tx == nil {
		return fmt.Errorf("invalid transaction")
	}

	// update submitter's DAG
	if err := e.db.UpdateSubmitter(tx); err != nil {
		return err
	}

	return nil
}

func NewEndorser(db repo.DltDb) (*endorser, error) {
	return &endorser{
		db: db,
	}, nil
}
