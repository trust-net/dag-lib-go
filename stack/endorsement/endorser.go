// Copyright 2018 The trust-net Authors
// Endorsement Layer interface and implementation for DLT Statck
package endorsement

import (
	"errors"
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

func (e *endorser) Anchor(a *dto.Anchor) error {
	return errors.New("not implemented")
}

func (e *endorser) Handle(tx dto.Transaction) error {
	// validate transaction
	// TBD
	if tx == nil {
		return errors.New("invalid transaction")
	}

	if err := e.db.AddTx(tx); err != nil {
		return err
	}

	// broadcast transaction
	// ^^^ this will be done by the controller if there is no error

	return nil
}

func (e *endorser) Approve(tx dto.Transaction) error {
	// validate transaction
	if tx == nil {
		return errors.New("invalid transaction")
	}

	// TBD: sign the node's signature on transaction

	return nil
}

func NewEndorser(db repo.DltDb) (*endorser, error) {
	return &endorser{
		db: db,
	}, nil
}
