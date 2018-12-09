package endorsement

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var e Endorser
	var err error
	testDb := repo.NewMockDltDb()
	e, err = NewEndorser(testDb)
	if e.(*endorser) == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", e, err)
	}
	if e.(*endorser).db != testDb {
		t.Errorf("Layer does not have correct DB reference expected: %s, actual: %s", testDb, e.(*endorser).db)
	}
}

func TestTxHandler(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a mock transaction to endorser
	if err := e.Handle(dto.TestTransaction()); err != nil {
		t.Errorf("Transacton handling failed: %s", err)
	}

	// validate that DltDb's AddTx method was called
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

func TestTxApprover(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a mock transaction to endorser
	if err := e.Approve(dto.TestTransaction()); err != nil {
		t.Errorf("Transacton approval failed: %s", err)
	}

	// validate that DltDb's AddTx method was NOT called during approval
	if testDb.AddTxCallCount != 0 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

func TestTxHandlerSavesTransaction(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

	// send a transaction to endorser
	tx := dto.TestSignedTransaction("test payload")
	e.Handle(tx)

	// verify if transaction is saved into endorser's DB using Transaction's signature as key
	if present := e.db.GetTx(tx.Id()); present == nil {
		t.Errorf("Transacton handling did not save the transaction")
	}
}

func TestTxHandlerBadTransaction(t *testing.T) {
	testDb := repo.NewMockDltDb()
	e, _ := NewEndorser(testDb)

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

	// validate that DltDb's AddTx method was called two times
	if testDb.AddTxCallCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}
