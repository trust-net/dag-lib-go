package shard

import (
	"fmt"
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

func TestGenesis(t *testing.T) {
	// create 2 different shard genesis transactions
	gen1 := GenesisShardTx([]byte("shard 1"))
	gen2 := GenesisShardTx([]byte("shard 2"))

	// verify that they both have different transaction IDs
	if gen1.Id() == gen2.Id() {
		t.Errorf("Genesis transactions not unique!!!")
	}

}

func TestRegistration(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// register an app
	txHandler := func(tx dto.Transaction) error { return nil }

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

	// validate that DltDb's UpdateShard method was called to update DAG and Tips with first genesis transaction
	if testDb.UpdateShardCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.UpdateShardCount)
	}
}

// test that app registration gets a replay of existing transactions
func TestRegistrationReplay(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test payload")
	s.db.AddTx(tx)
	s.Handle(tx)

	// register an app using same shard as network transaction
	cbCalled := false
	txHandler := func(tx dto.Transaction) error { cbCalled = true; return nil }
	if err := s.Register(tx.Anchor().ShardId, txHandler); err != nil {
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
	txHandler := func(tx dto.Transaction) error { return nil }

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
	txHandler := func(tx dto.Transaction) error { return nil }
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

func TestAnchorRegistered(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// register an app
	txHandler := func(tx dto.Transaction) error { return nil }
	s.Register([]byte("test shard"), txHandler)

	// call sharder's anchor update
	a := dto.Anchor{}
	if err := s.Anchor(&a); err != nil {
		t.Errorf("Anchor update failed: %s", err)
	}

	// anchor should have registered app's ID
	if string(a.ShardId) != "test shard" {
		t.Errorf("Incorrect shard ID: %s", a.ShardId)
	}

	// anchor should have shard's 1st sequence (since no other transaction after genesis)
	if a.ShardSeq != 0x01 {
		t.Errorf("Incorrect shard Seq: %x", a.ShardSeq)
	}

	// anchor should have shard's genesis as parent (since no other transaction after genesis)
	if a.ShardParent != GenesisShardTx(a.ShardId).Id() {
		t.Errorf("Incorrect shard parent: %x", a.ShardParent)
	}
}

func TestAnchorUnregistered(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// call sharder's anchor update without app registration
	a := dto.Anchor{}
	if err := s.Anchor(&a); err == nil {
		t.Errorf("Anchor update failed to check app registration")
	}
}

func TestAnchorMultiTip(t *testing.T) {
	fmt.Printf("#######################\n")
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// register an app
	txHandler := func(tx dto.Transaction) error { return nil }
	s.Register([]byte("test shard"), txHandler)

	// add 2 child network transactions nodes for same parent as genesis
	child1, _ := SignedShardTransaction("child1")
	child2, _ := SignedShardTransaction("child2")
	if err := s.db.AddTx(child1); err != nil {
		t.Errorf("Failed to add child1: %s", err)
	}

	if err := s.Handle(child1); err != nil {
		t.Errorf("Failed to handle child1: %s", err)
	}
	if err := s.db.AddTx(child2); err != nil {
		t.Errorf("Failed to add child2: %s", err)
	}

	if err := s.Handle(child2); err != nil {
		t.Errorf("Failed to handle child2: %s", err)
	}

	// call sharder's anchor update
	a := dto.Anchor{}
	if err := s.Anchor(&a); err != nil {
		t.Errorf("Anchor update failed: %s", err)
	}

	// anchor should have shard's 2nd sequence (since 1st seq is the network transaction after genesis)
	if a.ShardSeq != 0x02 {
		t.Errorf("Incorrect shard Seq: %x", a.ShardSeq)
	}

	// anchor should have weight of all tip's sequence summation + 1
	if a.Weight != (1+1)+1 {
		t.Errorf("Incorrect shard weight: %x", a.Weight)
	}

	// anchor should have highest numeric tip from the two
	parent := child1.Id()
	uncle := child2.Id()
	if Numeric(parent[:]) < Numeric(uncle[:]) {
		parent, uncle = uncle, parent
	}
	if a.ShardParent != parent {
		t.Errorf("Incorrect shard parent: %x", a.ShardParent)
	}
	if len(a.ShardUncles) != 1 {
		t.Errorf("Incorrect shard uncle count: %d", len(a.ShardUncles))
	} else if a.ShardUncles[0] != uncle {
		t.Errorf("Incorrect shard uncle: %x", a.ShardUncles[0])
	}
}

// test behavior for handling 1st transaction of a shard from network
func TestHandlerUnregisteredFirstSeq(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// send a mock network transaction with shard seq 1 to sharder with no app registered
	tx, genesis := SignedShardTransaction("test payload")
	s.db.AddTx(tx)
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
	tx.Anchor().ShardParent = [64]byte{}
	tx.Anchor().ShardParent[0] = 0xff
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
	txHandler := func(tx dto.Transaction) error { called = true; return nil }
	s.Register(tx.Anchor().ShardId, txHandler)

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
	txHandler := func(tx dto.Transaction) error { called = true; return nil }
	s.Register([]byte(string(tx.Anchor().ShardId)+"extra"), txHandler)

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
	txHandler := func(tx dto.Transaction) error { called = true; return nil }
	s.Register(tx.Anchor().ShardId, txHandler)

	// send the mock transaction to sharder with missing shard ID in transaction
	tx.Anchor().ShardId = nil
	if err := s.Handle(tx); err == nil {
		t.Errorf("sharder did not check for missing shard ID")
	}

	// verify that callback did not get called
	if called {
		t.Errorf("Sharder did not filter invalid transaction")
	}
}

// test behavior for approving a transaction when not registered (should not happen)
func TestApproverUnregisteredFirstSeq(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	// send a network transaction for approval with no app registered
	tx, _ := SignedShardTransaction("test payload")
	if err := s.Approve(tx); err == nil {
		t.Errorf("Approval of transacton did not check for app registration")
	}

	// validate that DltDb's GetShardDagNode method was NOT called for unregistered approval
	if testDb.GetShardDagNodeCallCount != 0 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's AddTx method was NOT called
	if testDb.AddTxCallCount != 0 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

func TestApproverHappyPath(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb)

	tx, _ := SignedShardTransaction("test payload")

	// register an app for transaction's shard
	called := false
	txHandler := func(tx dto.Transaction) error { called = true; return nil }
	s.Register(tx.Anchor().ShardId, txHandler)
	testDb.Reset()

	// send the transaction to sharder for approval
	if err := s.Approve(tx); err != nil {
		t.Errorf("Transaction approval failed: %s", err)
	}

	// verify that callback did not get called for submitted transaction
	if called {
		t.Errorf("Callback not expected for application submitted transaction")
	}

	// verify that DLT DB's shard was updated for submitted transaction
	if testDb.UpdateShardCount != 1 {
		t.Errorf("DLT DB's shard was NOT updated for submitted transaction: %d", testDb.UpdateShardCount)
	}

	// verify that submitted transaction was saved in DB
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Submitted transaction NOT saved in DB: %d", testDb.AddTxCallCount)
	}
}
