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

//func TestReplaySuccess(t *testing.T) {
//	e, _ := NewEndorser(db.NewInMemDbProvider())
//
//	// send couple of network transactions to endorser
//	tx1 := dto.TestSignedTransaction("test payload 1")
//	tx2 := dto.TestSignedTransaction("test payload 2")
//	if err := e.Handle(tx1); err != nil {
//		t.Errorf("Failed to submit transaction 1: %s\nid: %x", err, tx1.Id())
//	}
//	if err := e.Handle(tx2); err != nil {
//		t.Errorf("Failed to submit transaction 2: %s\nid: %x", err, tx2.Id())
//	}
//
//	// request a replay of transactions
//	callCount := 0
//	if err := e.Replay(func(tx *dto.Transaction) error {
//		callCount += 1
//		return nil
//	}); err != nil {
//		t.Errorf("Transaction replay failed: %s", err)
//	}
//
//	if callCount != 2 {
//		t.Errorf("Transaction replay only called back with %d message, expected: %d", callCount, 2)
//	}
//}
//
//func TestReplayError(t *testing.T) {
//	e, _ := NewEndorser(db.NewInMemDbProvider())
//
//	// send couple of network transactions to endorser
//	e.Handle(dto.TestSignedTransaction("test payload 1"))
//	e.Handle(dto.TestSignedTransaction("test payload 2"))
//
//	// request a replay of transactions
//	callCount := 0
//	if err := e.Replay(func(tx *dto.Transaction) error {
//		callCount += 1
//		return errors.New("forced error")
//	}); err == nil {
//		t.Errorf("Transaction replay did not abort upon error")
//	}
//
//	if callCount != 1 {
//		t.Errorf("Transaction replay called back with %d message, expected: %d", callCount, 1)
//	}
//}
//
//func TestReplayNoTransactions(t *testing.T) {
//	e, _ := NewEndorser(db.NewInMemDbProvider())
//
//	// request a replay of transactions
//	callCount := 0
//	if err := e.Replay(func(tx *dto.Transaction) error {
//		callCount += 1
//		return errors.New("forced error")
//	}); err != nil {
//		t.Errorf("Transaction replay failed: %s", err)
//	}
//
//	if callCount != 0 {
//		t.Errorf("Transaction replay called back with %d message, expected: %d", callCount, 0)
//	}
//}
