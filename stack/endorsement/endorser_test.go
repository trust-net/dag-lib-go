package endorsement

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
)

func TestInitiatization(t *testing.T) {
	var e Endorser
	var err error
	e, err = NewEndorser(db.NewInMemDatabase())
	if e.(*endorser) == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", e, err)
	}
}

func TestTxHandler(t *testing.T) {
	e, _ := NewEndorser(db.NewInMemDatabase())

	// send a mock transaction to endorser
	if err := e.Handle(dto.TestTransaction()); err != nil {
		t.Errorf("Transacton handling failed: %s", err)
	}
}

func TestTxHandlerSavesTransaction(t *testing.T) {
	e, _ := NewEndorser(db.NewInMemDatabase())

	// send a transaction to endorser
	tx := dto.TestSignedTransaction("test payload")
	e.Handle(tx)
	
	// verify if transaction is saved into endorser's DB using Transaction's signature as key
	if present,_ := e.db.Has(tx.Signature); !present {
		t.Errorf("Transacton handling did not save the transaction")
	}
}

func TestTxHandlerBadTransaction(t *testing.T) {
	e, _ := NewEndorser(db.NewInMemDatabase())

	// send a nil transaction to endorser
	if err := e.Handle(nil); err == nil {
		t.Errorf("Transacton handling did not check for nil transaction")
	}

	// send a duplicate transaction to endorser
	tx1 := dto.TestSignedTransaction("test payload")
	e.Handle(tx1)
	if err := e.Handle(tx1); err == nil {
		t.Errorf("Transacton handling did not check for duplicate transaction")
	}
}