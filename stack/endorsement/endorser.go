// Copyright 2018 The trust-net Authors
// Endorsement Layer interface and implementation for DLT Statck
package endorsement

import (
	"errors"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
)

type Endorser interface {
	// Handle Transaction
	Handle(tx *dto.Transaction) error
//	// Replay transactions for newly registered app
//	Replay(txHandler func(tx *dto.Transaction) error) error
//  let sharding layer handle replay using the new DltDb, no point processing ALL tx's, only need to process shard's DAG
}

type endorser struct {
	db repo.DltDb
}

func (e *endorser) Handle(tx *dto.Transaction) error {
	// validate transaction
	// TBD
	if tx == nil {
		return errors.New("invalid transaction")
	}

//	// check for duplicate transaction
//	if present, _ := e.db.Has(tx.Id()); present {
//		return errors.New("duplicate transaction")
//	}
//
//	// save transaction
//	var data []byte
//	var err error
//	if data, err = tx.Serialize(); err != nil {
//		return err
//	}
//	if err = e.db.Put(tx.Id(), data); err != nil {
//		return err
//	}
	if err := e.db.AddTx(tx); err != nil {
		return err
	}

	// broadcast transaction
	// ^^^ this will be done by the controller if there is no error

	return nil
}

//func (e *endorser) Replay(txHandler func(tx *dto.Transaction) error) error {
//	// get all transactions from DB and process each of them
//	for _, data := range e.db.GetAll() {
//		// deserialize the transaction read from DB
//		tx := &dto.Transaction{}
//		if err := tx.DeSerialize(data); err != nil {
//			return err
//		}
//		// process the transaction via callback
//		if err := txHandler(tx); err != nil {
//			return err
//		}
//	}
//	return nil
//}

func NewEndorser(db repo.DltDb) (*endorser, error) {
	return &endorser{
		db: db,
	}, nil
}
