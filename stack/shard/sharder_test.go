// Copyright 2018-2019 The trust-net Authors
package shard

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/state"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var s Sharder
	var err error
	testDb := repo.NewMockDltDb()
	s, err = NewSharder(testDb, db.NewInMemDbProvider())
	if s.(*sharder) == nil || err != nil {
		t.Errorf("Initiatization validation failed: %s, err: %s", s, err)
	}
	if s.(*sharder).db != testDb {
		t.Errorf("Layer does not have correct DB reference expected: %s, actual: %s", testDb, s.(*sharder).db)
	}
	if s.(*sharder).worldState != nil {
		t.Errorf("Sharder should initialize with nil world state, until an app is registered")
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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }

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

	// validate that DltDb's GetShardDagNode method was called twice for genesis node
	// first time to when there was no entry, second time after entry was created
	if testDb.GetShardDagNodeCallCount != 2 {
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

	// make sure that world state reference is nil (will be set then locked)
	if s.worldState != nil {
		t.Errorf("Sharder should not set world state reference when app is registered")
	}
}

// test that app registration gets a replay of existing transactions
func TestRegistrationReplay(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test payload")
	s.db.AddTx(tx)
	s.LockState()
	s.Handle(tx)
	s.CommitState(tx)
	s.UnlockState()

	// register an app using same shard as network transaction
	cbCalled := false
	txHandler := func(tx dto.Transaction, state state.State) error { cbCalled = true; return nil }
	if err := s.Register(tx.Request().ShardId, txHandler); err != nil {
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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// submit a network transaction for a shard

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }

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
	if testDb.GetShardDagNodeCallCount != 2 {
		t.Errorf("Incorrect method call count: %d", testDb.GetShardDagNodeCallCount)
	}

	// validate that DltDb's AddTx method was called to save the first genesis transaction
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Incorrect method call count: %d", testDb.AddTxCallCount)
	}
}

func TestUnregistration(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
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

	// make sure that world state reference is removed correctly
	if s.worldState != nil {
		t.Errorf("Sharder did not remove world state reference when app is unregistered")
	}
}

func TestAnchorRegistered(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register([]byte("test shard"), txHandler)
	testDb.Reset()

	// call sharder's anchor update
	a := dto.Anchor{}
	if err := s.Anchor(&a); err != nil {
		t.Errorf("Anchor update failed: %s", err)
	}

	// anchor does not have shard Id anymore
	//	// anchor should have registered app's ID
	//	if string(a.ShardId) != "test shard" {
	//		t.Errorf("Incorrect shard ID: %s", a.ShardId)
	//	}

	// anchor should have shard's 1st sequence (since no other transaction after genesis)
	if a.ShardSeq != 0x01 {
		t.Errorf("Incorrect shard Seq: %x", a.ShardSeq)
	}

	// anchor should have shard's genesis as parent (since no other transaction after genesis)
	if a.ShardParent != GenesisShardTx([]byte("test shard")).Id() {
		t.Errorf("Incorrect shard parent: %x", a.ShardParent)
	}
}

func TestAnchorUnregistered(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// call sharder's anchor update without app registration
	a := dto.Anchor{}
	if err := s.Anchor(&a); err == nil {
		t.Errorf("Anchor update failed to check app registration")
	}
}

func TestSyncAnchorRegsiteredKnown(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register([]byte("test shard"), txHandler)
	testDb.Reset()

	// call sharder's sync anchor for same shard as registered
	if a := s.SyncAnchor([]byte("test shard")); a == nil {
		t.Errorf("failed to get sync anchor for registered shard")
	}

	// we should not have created a genesis TX for the shard since its already known from before
	if testDb.AddTxCallCount != 0 {
		t.Errorf("should not create genesis transaction for known shard")
	} else if testDb.UpdateShardCount != 0 {
		t.Errorf("should not update shard DAG for genesis transaction of known shard")
	}
}

func TestSyncAnchorRegsiteredUnknown(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register([]byte("test shard"), txHandler)
	testDb.Reset()

	// call sharder's sync anchor for some unknown shard
	if a := s.SyncAnchor([]byte("unknown shard")); a != nil {
		t.Errorf("should not get sync anchor for unknown shard")
	}

	// however, we should have created a genesis TX for the shard, so that sync can happen
	if testDb.AddTxCallCount != 1 {
		t.Errorf("did not create genesis transaction for unknown shard")
	} else if tx := testDb.GetTx(GenesisShardTx([]byte("test shard")).Id()); tx == nil {
		t.Errorf("created incorrect genesis transaction for unknown shard")
	} else if testDb.UpdateShardCount != 1 || testDb.GetShardDagNode(tx.Id()) == nil {
		t.Errorf("did not update shard DAG for genesis transaction of unknown shard")
	}
}

func TestSyncAnchorUnregsiteredKnown(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register([]byte("test shard"), txHandler)
	testDb.Reset()

	// unregister the app
	s.Unregister()

	// call sharder's sync anchor for shard that is known from earlier
	if a := s.SyncAnchor([]byte("test shard")); a == nil {
		t.Errorf("failed to get sync anchor for known shard")
	}

	// we should not have created a genesis TX for the shard since its already known from before
	if testDb.AddTxCallCount != 0 {
		t.Errorf("should not create genesis transaction for known shard")
	} else if testDb.UpdateShardCount != 0 {
		t.Errorf("should not update shard DAG for genesis transaction of known shard")
	}
}

func TestSyncAnchorUnregsiteredUnknown(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// call sharder's sync anchor without app registration
	if a := s.SyncAnchor([]byte("unknown shard")); a != nil {
		t.Errorf("should not get sync anchor for unknown shard")
	}

	// however, we should have created a genesis TX for the shard, so that sync can happen
	if testDb.AddTxCallCount != 1 {
		t.Errorf("did not create genesis transaction for unknown shard")
	} else if tx := testDb.GetTx(GenesisShardTx([]byte("unknown shard")).Id()); tx == nil {
		t.Errorf("created incorrect genesis transaction for unknown shard")
	} else if testDb.UpdateShardCount != 1 || testDb.GetShardDagNode(tx.Id()) == nil {
		t.Errorf("did not update shard DAG for genesis transaction of unknown shard")
	}
}

func TestAnchorMultiTip(t *testing.T) {
	fmt.Printf("#######################\n")
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// register an app
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register([]byte("test shard"), txHandler)
	testDb.Reset()

	// add 2 child network transactions nodes for same parent as genesis
	child1, _ := SignedShardTransaction("child1")
	child2, _ := SignedShardTransaction("child2")
	s.db.AddTx(child1)
	s.LockState()
	s.Handle(child1)
	s.CommitState(child1)
	s.UnlockState()

	s.db.AddTx(child2)
	s.LockState()
	s.Handle(child2)
	s.CommitState(child2)
	s.UnlockState()

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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx, _ := SignedShardTransaction("test payload")

	// register an app for transaction's shard
	called := false
	txHandler := func(tx dto.Transaction, state state.State) error { called = true; return nil }
	s.Register(tx.Request().ShardId, txHandler)
	testDb.Reset()

	// send the mock network transaction to sharder with app registered
	s.LockState()
	defer s.UnlockState()
	if err := s.Handle(tx); err != nil {
		t.Errorf("Registered transacton handling failed: %s", err)
	}

	// verify that callback got called
	if !called {
		t.Errorf("Sharder did not invoke transaction call back")
	}

	// confirm the shard DAG update is not called yet (moved to commit stage)
	if testDb.UpdateShardCount != 0 {
		t.Errorf("Unexpected shard DAG update")
	}
}

func TestHandlerAppFiltering(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx, _ := SignedShardTransaction("test payload")

	// register an app for shard different from network transaction
	called := false
	txHandler := func(tx dto.Transaction, state state.State) error { called = true; return nil }
	s.Register([]byte(string(tx.Request().ShardId)+"extra"), txHandler)

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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx, _ := SignedShardTransaction("test payload")

	// register an app for transaction's shard
	called := false
	txHandler := func(tx dto.Transaction, state state.State) error { called = true; return nil }
	s.Register(tx.Request().ShardId, txHandler)

	// send the mock transaction to sharder with missing shard ID in transaction
	tx.Request().ShardId = nil
	s.LockState()
	defer s.UnlockState()
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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

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
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx, _ := SignedShardTransaction("test payload")

	// register an app for transaction's shard
	called := false
	txHandler := func(tx dto.Transaction, state state.State) error { called = true; return nil }
	s.Register(tx.Request().ShardId, txHandler)
	testDb.Reset()

	// send the transaction to sharder for approval
	s.LockState()
	defer s.UnlockState()
	if err := s.Approve(tx); err != nil {
		t.Errorf("Transaction approval failed: %s", err)
	}

	// verify that callback did get called for submitted transaction
	if !called {
		t.Errorf("Callback not done for application submitted transaction")
	}

	// confirm the shard DAG update is not called yet (moved to commit stage)
	if testDb.UpdateShardCount != 0 {
		t.Errorf("Unexpected shard DAG update")
	}

	// verify that submitted transaction was saved in DB
	if testDb.AddTxCallCount != 1 {
		t.Errorf("Submitted transaction NOT saved in DB: %d", testDb.AddTxCallCount)
	}
}

func TestAncestorsKnownStartHash(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx1, genesis := SignedShardTransaction("test payload")
	tx2 := dto.TestSignedTransaction("test payload")
	tx2.Anchor().ShardParent = tx1.Id()
	tx2.Anchor().ShardSeq = tx1.Anchor().ShardSeq + 1
	// register an app for transaction's shard
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register(tx1.Request().ShardId, txHandler)

	// add transactions to sharder's DAG
	s.LockState()
	if err := s.Handle(tx1); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	s.CommitState(tx1)
	s.UnlockState()

	s.LockState()
	if err := s.Handle(tx2); err != nil {
		t.Errorf("Failed to add 2nd transaction: %s", err)
	}
	s.CommitState(tx2)
	s.UnlockState()

	// now fetch ancestors from tx2 as starting hash
	ancestors := s.Ancestors(tx2.Id(), 5)

	// we should get 2 ancestors: tx1 and genesis
	if len(ancestors) != 2 {
		t.Errorf("Incorrect number of ancestors: %d", len(ancestors))
	} else if ancestors[0] != tx1.Id() {
		t.Errorf("Incorrect 1st ancestor:\n%x\nExpected:\n%x", ancestors[0], tx1.Id())
	} else if ancestors[1] != genesis.Id() {
		t.Errorf("Incorrect 1st ancestor:\n%x\nExpected:\n%x", ancestors[1], genesis.Id())
	}
}

func TestAncestorsUnknownStartHash(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx1, _ := SignedShardTransaction("test payload")
	tx2 := dto.TestSignedTransaction("test payload")
	tx2.Anchor().ShardParent = tx1.Id()
	tx2.Anchor().ShardSeq = tx1.Anchor().ShardSeq + 1
	// register an app for transaction's shard
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register(tx1.Request().ShardId, txHandler)

	// add transactions to sharder's DAG
	s.LockState()
	if err := s.Handle(tx1); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	s.CommitState(tx1)
	s.UnlockState()

	s.LockState()
	if err := s.Handle(tx2); err != nil {
		t.Errorf("Failed to add 2nd transaction: %s", err)
	}
	s.CommitState(tx2)
	s.UnlockState()

	// now fetch ancestors from an unknown starting hash
	hash := tx2.Id()
	hash[5] = 0x00
	hash[6] = 0x00
	hash[7] = 0x00
	ancestors := s.Ancestors(hash, 5)

	// we should get 0 ancestors
	if len(ancestors) != 0 {
		t.Errorf("Incorrect number of ancestors: %d", len(ancestors))
	}
}

func TestChildrenKnownParent(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx1, _ := SignedShardTransaction("test payload")
	tx2 := dto.TestSignedTransaction("test payload")
	tx2.Anchor().ShardParent = tx1.Id()
	tx2.Anchor().ShardSeq = tx1.Anchor().ShardSeq + 1
	// register an app for transaction's shard
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register(tx1.Request().ShardId, txHandler)

	// add transactions to sharder's DAG
	s.LockState()
	if err := s.Handle(tx1); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	s.CommitState(tx1)
	s.UnlockState()

	s.LockState()
	if err := s.Handle(tx2); err != nil {
		t.Errorf("Failed to add 2nd transaction: %s", err)
	}
	s.CommitState(tx2)
	s.UnlockState()

	// now fetch children for tx1 as parent
	children := s.Children(tx1.Id())

	// we should get 1 child: tx2
	if len(children) != 1 {
		t.Errorf("Incorrect number of children: %d", len(children))
	} else if children[0] != tx2.Id() {
		t.Errorf("Incorrect 1st child:\n%x\nExpected:\n%x", children[0], tx2.Id())
	}
}

func TestChildrenUnknownParent(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	tx1, _ := SignedShardTransaction("test payload")
	tx2 := dto.TestSignedTransaction("test payload")
	tx2.Anchor().ShardParent = tx1.Id()
	tx2.Anchor().ShardSeq = tx1.Anchor().ShardSeq + 1
	// register an app for transaction's shard
	txHandler := func(tx dto.Transaction, state state.State) error { return nil }
	s.Register(tx1.Request().ShardId, txHandler)

	// add transactions to sharder's DAG
	s.LockState()
	if err := s.Handle(tx1); err != nil {
		t.Errorf("Failed to add 1st transaction: %s", err)
	}
	s.CommitState(tx1)
	s.UnlockState()

	s.LockState()
	if err := s.Handle(tx2); err != nil {
		t.Errorf("Failed to add 2nd transaction: %s", err)
	}
	s.CommitState(tx2)
	s.UnlockState()

	// now fetch children from an unknown parent
	hash := tx1.Id()
	hash[5] = 0x00
	hash[6] = 0x00
	hash[7] = 0x00
	children := s.Children(hash)

	// we should get 0 child
	if len(children) != 0 {
		t.Errorf("Incorrect number of children: %d", len(children))
	}
}

func TestWorldStateResourceAccess(t *testing.T) {
	testDb := repo.NewMockDltDb()
	dbp := db.NewInMemDbProvider()
	s, _ := NewSharder(testDb, dbp)

	// register an app
	txHandler := func(tx dto.Transaction, s state.State) error {
		if r, err := s.Get([]byte("key")); err != nil {
			return err
		} else if string(r.Value) != string(tx.Request().Payload) {
			return fmt.Errorf("Unexpected resource value: '%s', expected: %s", r.Value, tx.Request().Payload)
		}
		return nil
	}

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test data to validate")
	s.db.AddTx(tx)
	s.Handle(tx)
	testShard := tx.Request().ShardId

	// set test value in world state for test shard
	db := dbp.DB("Shard-World-State-" + string(testShard))
	r := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test data to validate"),
	}
	data, _ := r.Serialize()
	db.Put(r.Key, data)

	// now register the app for the shard and it should check for above world state value
	if err := s.Register(testShard, txHandler); err != nil {
		t.Errorf("%s", err)
	}
}

func TestWorldStateResourceVisibilityAcrossShards(t *testing.T) {
	testDb := repo.NewMockDltDb()
	dbp := db.NewInMemDbProvider()
	s, _ := NewSharder(testDb, dbp)

	// register an app
	var seenResource *state.Resource
	txHandler := func(tx dto.Transaction, s state.State) error {
		seenResource, _ = s.Get(tx.Request().Payload)
		return nil
	}

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("key")
	s.db.AddTx(tx)
	s.Handle(tx)
	testShard := tx.Request().ShardId

	// set test value in world state for test shard
	db := dbp.DB("Shard-World-State-" + string(testShard))
	r := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test shard 1"),
	}
	data, _ := r.Serialize()
	db.Put(r.Key, data)

	// now register the app for the shard and it should check for above world state value
	if err := s.Register(testShard, txHandler); err != nil {
		t.Errorf("%s", err)
	}

	// validate that resource was found
	if r == nil || string(r.Value) != "test shard 1" {
		t.Errorf("Found unexpected resource: %s", r)
	}

	// unregister app
	s.Unregister()

	// put a different value for resource in a different shard
	r2 := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test shard 2"),
	}
	data2, _ := r2.Serialize()
	dbp.DB("Shard-World-State-"+string([]byte("some random shard"))).Put(r2.Key, data2)

	// now register the app again
	if err := s.Register(testShard, txHandler); err != nil {
		t.Errorf("%s", err)
	}

	// validate that resource was not found this time with other shard's value
	if r == nil || string(r.Value) == "test shard 2" {
		t.Errorf("Found unexpected resource: %s", r)
	}
}

func TestWorldStateReadAccessRegisteredApp(t *testing.T) {
	testDb := repo.NewMockDltDb()
	dbp := db.NewInMemDbProvider()
	s, _ := NewSharder(testDb, dbp)

	// register an app
	txHandler := func(tx dto.Transaction, s state.State) error { return nil }

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test data to validate")
	s.db.AddTx(tx)
	s.Handle(tx)
	testShard := tx.Request().ShardId

	// set test value in world state for test shard
	db := dbp.DB("Shard-World-State-" + string(testShard))
	r := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test data to validate"),
	}
	data, _ := r.Serialize()
	db.Put(r.Key, data)

	// now register the app for the shard
	s.Register(testShard, txHandler)

	// lookup resource value using read API
	if read, err := s.GetState([]byte("key")); err != nil {
		t.Errorf("Failed to get state: %s", err)
	} else if string(read.Value) != "test data to validate" {
		t.Errorf("Incorrect data from get state: %s", read.Value)
	}
}

func TestWorldStateReadAccessNoApp(t *testing.T) {
	testDb := repo.NewMockDltDb()
	dbp := db.NewInMemDbProvider()
	s, _ := NewSharder(testDb, dbp)

	// register an app
	txHandler := func(tx dto.Transaction, s state.State) error { return nil }

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test data to validate")
	s.db.AddTx(tx)
	s.Handle(tx)
	testShard := tx.Request().ShardId

	// set test value in world state for test shard
	db := dbp.DB("Shard-World-State-" + string(testShard))
	r := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test data to validate"),
	}
	data, _ := r.Serialize()
	db.Put(r.Key, data)

	// register the app for the shard so that it processed transaction
	s.Register(testShard, txHandler)

	// now un register the app from the shard
	s.Unregister()

	// lookup resource value using read API, we should get error since no app registered
	if _, err := s.GetState([]byte("key")); err == nil {
		t.Errorf("Should fail to get state for unregistered app")
	}
}

// test shard flush when app is registered
func TestFlush_RegisteredApp(t *testing.T) {
	testDb := repo.NewMockDltDb()
	dbp := db.NewInMemDbProvider()
	s, _ := NewSharder(testDb, dbp)

	// register an app
	txHandler := func(tx dto.Transaction, s state.State) error { return nil }

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test data to validate")
	s.db.AddTx(tx)
	s.Handle(tx)
	testShard := tx.Request().ShardId

	// set test value in world state for test shard
	db := dbp.DB("Shard-World-State-" + string(testShard))
	r := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test data to validate"),
	}
	data, _ := r.Serialize()
	db.Put(r.Key, data)

	// now register the app for the shard
	s.Register(testShard, txHandler)

	// lookup resource value using read API
	if read, err := s.GetState([]byte("key")); err != nil {
		t.Errorf("Failed to get state: %s", err)
	} else if string(read.Value) != "test data to validate" {
		t.Errorf("Incorrect data from get state: %s", read.Value)
	}

	// now flush the shard
	if err := s.Flush(testShard); err != nil {
		t.Errorf("shard flush failed: %s", err)
	}

	// validate the world state was flushed
	if _, err := s.GetState([]byte("key")); err == nil {
		t.Errorf("did not expect to get resource after flush")
	}
}

// test shard flush for a shard different from registered app
func TestFlush_UnregisteredShard(t *testing.T) {
	testDb := repo.NewMockDltDb()
	dbp := db.NewInMemDbProvider()
	s, _ := NewSharder(testDb, dbp)

	// register an app
	txHandler := func(tx dto.Transaction, s state.State) error { return nil }

	// send a mock network transaction with shard seq 1 to sharder before app is registered
	tx, _ := SignedShardTransaction("test data to validate")
	s.db.AddTx(tx)
	s.Handle(tx)
	testShard := tx.Request().ShardId

	// set test value in world state for test shard
	db := dbp.DB("Shard-World-State-" + string(testShard))
	r := &state.Resource{
		Key:   []byte("key"),
		Value: []byte("test data to validate"),
	}
	data, _ := r.Serialize()
	db.Put(r.Key, data)

	// now register the app for the shard
	s.Register(testShard, txHandler)

	// now flush the shard for some other id
	if err := s.Flush([]byte("a different shard")); err != nil {
		t.Errorf("shard flush failed: %s", err)
	}

	// lookup resource value using read API for registered app's shard
	// flush of different shard should not have affected registered app
	if read, err := s.GetState([]byte("key")); err != nil {
		t.Errorf("Failed to get state: %s", err)
	} else if string(read.Value) != "test data to validate" {
		t.Errorf("Incorrect data from get state: %s", read.Value)
	}
}

func TestCommitState_WithTransaction(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())
	tx, _ := SignedShardTransaction("test payload")

	// call commit state with a transaction (e.g. after submitted transaction approval
	// or after network transaction handling)
	s.CommitState(tx)

	// confirm that shard DAG was updated
	if testDb.UpdateShardCount != 1 {
		t.Errorf("Commit state did not update shard DAG")
	}
}

func TestCommitState_NilTransaction(t *testing.T) {
	testDb := repo.NewMockDltDb()
	s, _ := NewSharder(testDb, db.NewInMemDbProvider())

	// call commit state with nil transaction (e.g. after app registration replay)
	s.CommitState(nil)

	// confirm that shard DAG was not updated
	if testDb.UpdateShardCount != 0 {
		t.Errorf("Commit state should not update shard DAG")
	}
}
