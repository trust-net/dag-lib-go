package shard

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var s Sharder
	var err error
	testDb := repo.NewMockDltDb()
	s, err = NewSharder(testDb)
	if s.(*sharder) == nil || err != nil {
		t.Errorf("Initiatization validation failed: %s, err: %s", s, err)
	}
	if s.(*sharder).db != testDb {
		t.Errorf("Layer does not have correct DB reference expected: %s, actual: %s", testDb, s.(*sharder).db)
	}
}

func TestRegistration(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// register an app
	txHandler := func(tx *dto.Transaction) error { return nil }

	if err := s.Register([]byte("test shard"), txHandler); err != nil {
		t.Errorf("App registration failed: %s", err)
	}

	// make sure sharder registered the values
	if s.shardId == nil {
		t.Errorf("Sharder did not register app's shard ID")
	}
	if s.txHandler == nil {
		t.Errorf("Sharder did not register transaction call back")
	}

	// validate that DltDb's GetShardDagNode method was called for genesis node
	if testDb.GetShardDagNodeCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's AddTx method was called to save the first genesis transaction
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

// test that app registration gets a replay of existing transactions
func TestRegistrationReplay(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test payload")
	s.Handle(tx)

	// register an app using same shard as network transaction
	cbCalled := false
	txHandler := func(tx *dto.Transaction) error { cbCalled = true; return nil }
	if err := s.Register(tx.ShardId, txHandler); err != nil {
		t.Errorf("App registration failed: %s", err)
	}

	// replay should have called application's transaction handler
	if !cbCalled {
		t.Errorf("App registration did not replay transactions to the app")
	}

	// validate that DltDb's GetShardDagNode method was called 3 times:
	//   1) for genesis node's parent during saving genesis node
	//   2) for reading genesis node at the beginning of replay
	//   3) for reading network transaction node as child of genesis node during replay
	if testDb.GetShardDagNodeCallCount != 3 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's GetTx method was called to replay the network transaction
	if testDb.GetTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.GetTxCallCount)
	}
}

func TestRegistrationKnownShard(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// submit a network transaction for a shard

	// register an app
	txHandler := func(tx *dto.Transaction) error { return nil }

	if err := s.Register([]byte("test shard"), txHandler); err != nil {
		t.Errorf("App registration failed: %s", err)
	}

	// make sure sharder registered the values
	if s.shardId == nil {
		t.Errorf("Sharder did not register app's shard ID")
	}
	if s.txHandler == nil {
		t.Errorf("Sharder did not register transaction call back")
	}

	// validate that DltDb's GetShardDagNode method was called for genesis node
	if testDb.GetShardDagNodeCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's AddTx method was called to save the first genesis transaction
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

func TestUnregistration(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// register an app
	txHandler := func(tx *dto.Transaction) error { return nil }
	s.Register([]byte("test shard"), txHandler)

	// un-register the app
	if err := s.Unregister(); err != nil {
		t.Errorf("App un-registration failed: %s", err)
	}

	// make sure sharder cleared the values
	if s.shardId != nil {
		t.Errorf("Sharder did not clear app's shard ID")
	}
	if s.txHandler != nil {
		t.Errorf("Sharder did not clear transaction call back")
	}
}

// test behavior for handling 1st transaction of a shard from network
func TestHandlerUnregisteredFirstSeq(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// send a mock network transaction with shard seq 1 to sharder with no app registered
	tx, genesis := SignedShardTransaction("test payload")
	if err := s.Handle(tx); err != nil {
		t.Errorf("Network handling of 1st shard transacton failed: %s", err)
	}

	// validate that a genesis transaction was added for the 1st seq of unknown shard
	if gen := testDb.GetTx(genesis.Id()); gen == nil {
		t.Errorf("Sharder did not create genesis transaction for 1st seq of unknown shard")
	}

	// validate that DltDb's GetShardDagNode method was called for genesis node
	if testDb.GetShardDagNodeCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's AddTx method was called twice:
	//    1) save the first genesis transaction
	//    2) to save the network transaction
	if testDb.AddTxCallCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

// test behavior for handling incorrect 1st transaction of a shard from network
func TestHandlerIncorrectGenesisFirstSeq(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// send a mock network transaction with shard seq 1 but incorrect parent not matching correct genesis
	tx, genesis := SignedShardTransaction("test payload")
	tx.ShardParent = [64]byte{}
	tx.ShardParent[0]=0xff
	if err := s.Handle(tx); err == nil {
		t.Errorf("Network handling of 1st shard transacton did not validate genesis parent")
	}

	// validate that a genesis transaction was not added for the 1st seq of unknown shard
	if gen := testDb.GetTx(genesis.Id()); gen != nil {
		t.Errorf("Sharder not expected to create genesis transaction for 1st seq of incorrect parent")
	}

	// validate that DltDb's GetShardDagNode method was not called for genesis node
	if testDb.GetShardDagNodeCallCount != 0 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's AddTx method was not called at all since parent genesis does not match
	if testDb.AddTxCallCount != 0 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

func TestHandlerRegistered(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	tx, _ := SignedShardTransaction("test payload")

	// register an app for transaction's shard
	called := false
	txHandler := func(tx *dto.Transaction) error { called = true; return nil }
	s.Register(tx.ShardId, txHandler)

	// send the mock network transaction to sharder with app registered
	if err := s.Handle(tx); err != nil {
		t.Errorf("Registered transacton handling failed: %s", err)
	}

	// verify that callback got called
	if !called {
		t.Errorf("Sharder did not invoke transaction call back")
	}
}

func TestHandlerAppFiltering(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	tx, _ := SignedShardTransaction("test payload")

	// register an app for shard different from network transaction
	called := false
	txHandler := func(tx *dto.Transaction) error { called = true; return nil }
	s.Register([]byte(string(tx.ShardId) + "extra"), txHandler)

	// send the mock network transaction to sharder from different shard
	if err := s.Handle(tx); err != nil {
		t.Errorf("Unregistered transacton handling failed: %s", err)
	}

	// verify that callback did not get called
	if called {
		t.Errorf("Sharder did not filter transaction from unregistered shard")
	}
}

func TestHandlerTransactionValidation(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	tx, _ := SignedShardTransaction("test payload")

	// register an app for transaction's shard
	called := false
	txHandler := func(tx *dto.Transaction) error { called = true; return nil }
	s.Register(tx.ShardId, txHandler)

	// send the mock transaction to sharder with missing shard ID in transaction
	tx.ShardId = nil
	if err := s.Handle(&dto.Transaction{}); err == nil {
		t.Errorf("sharder did not check for missing shard ID")
	}

	// verify that callback did not get called
	if called {
		t.Errorf("Sharder did not filter invalid transaction")
	}
}
