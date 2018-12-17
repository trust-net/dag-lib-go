package stack

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"testing"
	"time"
)

func initMocks() (*dlt, *mockSharder, *mockEndorser, *p2p.MockP2P) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// inject mock sharding layer into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder

	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser

	// register app
	app := TestAppConfig()
	stack.Register(app.ShardId, app.Name, func(tx dto.Transaction) error { return nil })

	return stack, sharder, endorser, mockP2PLayer
}

// initialize DLT stack and validate
func TestInitiatization(t *testing.T) {
	var stack DLT
	var err error
	testDb := db.NewInMemDbProvider()
	stack, err = NewDltStack(p2p.TestConfig(), testDb)
	if stack.(*dlt) == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", stack, err)
	}
	//	if stack.(*dlt).db != testDb.DB("dlt_stack") {
	//		t.Errorf("Stack does not have correct DB reference expected: %s, actual: %s", testDb.DB("dlt_stack").Name(), stack.(*dlt).db.Name())
	//	}
	if len(stack.(*dlt).p2p.Self()) == 0 {
		t.Errorf("Stack does not have correct p2p layer")
	}
	if stack.(*dlt).endorser == nil {
		t.Errorf("Stack does not have endorsement layer")
	}
	if stack.(*dlt).sharder == nil {
		t.Errorf("Stack does not have sharding layer")
	}
}

// register application
func TestRegister(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// register a transaction with sharder to be replayed upon app registration
	tx, _ := shard.SignedShardTransaction("test payload")
	endorser.Handle(tx)
	sharder.Handle(tx)

	// unregister default app and register a new app that remembers when replay transaction
	stack.Unregister()
	app := TestAppConfig()
	cbCalled := false
	txHandler := func(tx dto.Transaction) error { cbCalled = true; return nil }
	sharder.Reset()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed upon replay error: %s", err)
	}

	// our app's ID should be same as p2p node's ID
	if string(stack.app.AppId) != string(stack.p2p.Id()) || string(stack.app.ShardId) != string(tx.Anchor().ShardId) || stack.app.Name != app.Name {
		t.Errorf("App configuration not initialized correctly")
	}
	if stack.txHandler == nil {
		t.Errorf("Callback methods not initialized correctly")
	}

	// we should have registered with sharder
	if !sharder.IsRegistered || sharder.TxHandler == nil {
		t.Errorf("DLT stack controller did not register with sharding layer")
	}

	// replay should have called application's transaction handler
	if !cbCalled {
		t.Errorf("DLT stack app registration did not replay transactions to the app")
	}

}

// replay failure during register application
func TestRegisterReplayFailure(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// register a transaction with sharder to be replayed upon app registration
	tx, _ := shard.SignedShardTransaction("test payload")
	endorser.Handle(tx)
	sharder.Handle(tx)

	// unregister default app and register a new app that rejects replay transaction
	stack.Unregister()
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return errors.New("forced failure") }
	sharder.Reset()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed upon replay error: %s", err)
	}

	// we should be registered with sharder
	if !sharder.IsRegistered || sharder.TxHandler == nil {
		t.Errorf("DLT stack controller did not keep app registration with sharding layer upon replay failure")
	}
}

// attempt to register app when already registered
func TestPreRegistered(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

	// attempt to register app again
	sharder.Reset()
	txHandler := func(tx dto.Transaction) error { return nil }
	if err := stack.Register([]byte("another shard"), "another app", txHandler); err == nil {
		t.Errorf("Registration did not check for already registered")
	}

	// we should NOT have registered with sharder
	if sharder.IsRegistered || sharder.TxHandler != nil {
		t.Errorf("DLT stack controller did duplicate register with sharding layer")
	}
}

// unregister a previously registered application
func TestUnRegister(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

	// now unregister app
	if err := stack.Unregister(); err != nil {
		t.Errorf("Unregistration failed, err: %s", err)
	}
	if stack.app != nil {
		t.Errorf("App configuration not cleared during unregister")
	}
	if stack.txHandler != nil {
		t.Errorf("Callback methods not cleared during unregister")
	}

	// we should have un-registered with sharder and cleared the callback reference
	if sharder.IsRegistered || sharder.TxHandler != nil {
		t.Errorf("DLT stack controller did not unregister with sharding layer")
	}
}

// try submitting a transaction without application being registered first
func TestSubmitUnregistered(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for unregistered")
	}
	// make sure that endorser does not gets called for unregistered submission
	// since its lower in stack from sharder, which would have errored out already
	if endorser.TxHandlerCalled {
		t.Errorf("Endorser called for unregistered submission")
	}
}

// try submitting a transaction with nil/missing values
func TestSubmitNilValues(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return nil }
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}

	// try submitting a nil transaction
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for nil transaction")
	}

	// try submitting nil payload
	tx := TestTransaction()
	tx.Self().Payload = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil payload")
	}

	// try submitting unsigned transaction
	tx = TestTransaction()
	tx.Self().Signature = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for signature")
	}

	// submitter ID needs to be non-null
	tx = TestTransaction()
	tx.Anchor().Submitter = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil submitter ID")
	}
}

// try submitting a transaction with fake app ID, it should fail
func TestSubmitAppIdNoMatch(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return nil }
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
}

// transaction submission, happy path
func TestSubmit(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2p := initMocks()

	// get an anchor
	a := stack.Anchor([]byte("test submitter"))
	if a == nil {
		t.Errorf("Failed to get anchor")
	}

	tx := TestAnchoredTransaction(a, "test payload")
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}

	// walk down the DLT stack and validate expectations

	// validate sharding layer
	// verify that sharding layer gets called for submission
	if !sharder.ApproverCalled {
		t.Errorf("Sharder did not get called for submission")
	}

	// verify that endorser gets called for submission
	if !endorser.ApproverCalled {
		t.Errorf("Endorser did not get called for submission")
	}
	// verify that endorser got the right transaction
	if endorser.TxId != tx.Id() {
		t.Errorf("Endorser transaction does not match submitted transaction")
	}

	// validate p2p layer
	if !p2p.DidBroadcast {
		t.Errorf("Transaction did not get broadcast to peers")
	}
	// verify that transaction's node ID was set correctly
	if string(tx.Anchor().NodeId) != string(p2p.Id()) {
		t.Errorf("Transaction's node ID not initialized correctly\nExpected: %x\nActual: %x", p2p.Id(), tx.Anchor().NodeId)
	}
}

// transaction submission of a seen transaction
func TestReSubmitSeen(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// get an anchor
	a := stack.Anchor([]byte("test submitter"))
	if a == nil {
		t.Errorf("Failed to get anchor")
	}

	tx := TestAnchoredTransaction(a, "test payload")
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}

	// reset and re-submit same transaction
	sharder.Reset()
	endorser.Reset()
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction re-submission did not fail")
	}

	// verify that sharding layer does not gets called for seen submission
	if sharder.ApproverCalled {
		t.Errorf("Sharder should not get called for seen submission")
	}

	// verify that endorser does not gets called for seen submission
	if endorser.ApproverCalled {
		t.Errorf("Endorser should not get called for seen submission")
	}
}

// transaction submission of a seen transaction
func TestSubmitNetworkSeen(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// get an anchor
	a := stack.Anchor([]byte("test submitter"))
	if a == nil {
		t.Errorf("Failed to get anchor")
	}

	// simulate a submission of transaction that is already seen by stack
	tx := TestAnchoredTransaction(a, "test payload")
	stack.isSeen(tx.Id())

	if err := stack.Submit(tx); err == nil {
		t.Errorf("Seen transaction submission did not fail")
	}

	// verify that sharding layer does not gets called for seen submission
	if sharder.TxHandlerCalled {
		t.Errorf("Sharder should not get called for seen submission")
	}

	// verify that endorser does not gets called for seen submission
	if endorser.TxHandlerCalled {
		t.Errorf("Endorser should not get called for seen submission")
	}
}

// transaction submission validation of fields
func TestSubmitValidation(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return nil }

	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	tx := TestTransaction()
	tx.Anchor().ShardId = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for missing shard Id")
	}
	if p2p.DidBroadcast {
		t.Errorf("Invalid transaction got broadcast to peers")
	}
}

// start of controller, happy path
func TestStart(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	if err := stack.Start(); err != nil || !p2p.IsStarted {
		t.Errorf("Controller failed to start: %s", err)
	}
	if !p2p.IsStarted {
		t.Errorf("Controller did not start p2p layer")
	}
}

// stop of controller, happy path
func TestStop(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	stack.Stop()
	if !p2p.IsStopped {
		t.Errorf("Controller did not stop p2p layer")
	}
}

// get an anchor from DLT stack when app is registered
func TestAnchorRegisteredApp(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// get an anchor
	a := stack.Anchor([]byte("test submitter"))
	if a == nil {
		t.Errorf("Failed to get anchor")
	}

	if !sharder.AnchorCalled {
		t.Errorf("DLT stack did not called sharder's Anchor")
	}

	if !endorser.AnchorCalled {
		t.Errorf("DLT stack did not called endorser's Anchor")
	}
}

// get an anchor from DLT stack when app is not registered
func TestAnchorUnregisteredApp(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// unregister app
	stack.Unregister()

	// get an anchor
	a := stack.Anchor([]byte("test submitter"))
	if a != nil {
		t.Errorf("should not get anchor when app is not registered")
	}

	if !sharder.AnchorCalled {
		t.Errorf("DLT stack did not called sharder's Anchor")
	}

	if endorser.AnchorCalled {
		t.Errorf("DLT stack should not call endorser's Anchor when app is not registered")
	}
}

// peer connection handshake, happy path
func TestPeerHandshake(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// invoke handshake
	if err := stack.handshake(peer); err != nil {
		t.Errorf("Handshake failed, err: %s", err)
	}

	// we should have got anchor from sharding layer
	if !sharder.AnchorCalled {
		t.Errorf("Handshake did not fetch Anchor from sharding layer")
	}

	// we should have got anchor from endorsing layer
	if !endorser.AnchorCalled {
		t.Errorf("Handshake did not fetch Anchor from endorser layer")
	}

	// we should have sent ShardSyncMsg to peer
	if !peer.SendCalled {
		t.Errorf("Handshake did not send any message to peer")
	} else if peer.SendMsgCode != ShardSyncMsgCode {
		t.Errorf("Handshake did not send ShardSyncMsg message to peer")
	}
	if mockConn.WriteCount != 1 {
		t.Errorf("Handshake sent unexpected number of messages: %d", mockConn.WriteCount)
	}
}

// peer connection handshake when no app registered
func TestPeerHandshakeUnregistered(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// make sure no app is registered
	stack.Unregister()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// invoke handshake
	if err := stack.handshake(peer); err != nil {
		t.Errorf("Handshake failed, err: %s", err)
	}

	// we should have got anchor from sharding layer
	if !sharder.AnchorCalled {
		t.Errorf("Handshake did not fetch Anchor from sharding layer")
	}

	// we should not have got anchor from endorsing layer
	if endorser.AnchorCalled {
		t.Errorf("Handshake should not fetch Anchor from endorser layer for unregistered app")
	}

	// we should not have sent ShardSyncMsg to peer
	if peer.SendCalled {
		t.Errorf("Handshake should not send any message to peer for unregistered app")
	}
	if mockConn.WriteCount != 0 {
		t.Errorf("Handshake sent unexpected number of messages: %d", mockConn.WriteCount)
	}
}

// test stack controller event listener handles RECV_NewTxBlockMsg correctly
func TestEventListenerHandleRECV_NewTxBlockMsgEvent(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_NewTxBlockMsg event
	tx := TestSignedTransaction("test payload")
	events <- newControllerEvent(RECV_NewTxBlockMsg, tx)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle new transaction

	// we should have broadcasted message
	if !p2pLayer.DidBroadcast {
		t.Errorf("Listener did not froward network transaction as headless")
	}

	// sharding layer should be asked to handle transaction
	if !sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller did not call sharding layer")
	}

	// verify that endorser gets called for network message
	if !endorser.TxHandlerCalled {
		t.Errorf("Endorser did not get called for network transaction")
	}
	if endorser.TxId != tx.Id() {
		t.Errorf("Endorser transaction does not match network transaction")
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when remote weight is more
func TestRECV_ShardSyncMsgEvent_RemoteHeavy(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with heavier Anchor
	a := stack.Anchor([]byte("test submitter"))
	a.Weight += 10
	msg := NewShardSyncMsg(a)
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// when peer's Anchor is heavier than local shard's Anchor

	// we should have used sharder's shard sync anchor
	if !sharder.SyncAnchorCalled {
		t.Errorf("controller did not use sharder's shard sync anchor")
	}

	// we should have sent the ShardAncestorRequestMsg message from peer Anchor's parent to walk back on DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when local shard is heavy
func TestRECV_ShardSyncMsgEvent_LocalHeavy(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transaction to add weight to local shard's Anchor
	tx := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload")
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with default Anchor but same shard as local
	a := dto.TestAnchor()
	a.ShardId = stack.Anchor([]byte("test")).ShardId
	msg := NewShardSyncMsg(a)
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// since local Anchor is heavier than remote shard's Anchor, no further sync message should be sent

	// we should not have sent the ShardAncestorRequestMsg message
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when both weights are same
func TestRECV_ShardSyncMsgEvent_SameWeight(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with same weight but parent hash that is higher numeric value
	a := stack.Anchor([]byte("test submitter"))
	for i := 0; i < 64; i++ {
		a.ShardParent[i] = 0xff
	}
	msg := NewShardSyncMsg(a)
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// when peer's Anchor is heavier than local shard's Anchor

	// we should have sent the ShardAncestorRequestMsg message from peer Anchor's parent to walk back on DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when both shards have same anchor
func TestRECV_ShardSyncMsgEvent_SameAnchors(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transaction to add weight to local shard's Anchor
	tx := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload")
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with same anchor as local
	local := stack.Anchor([]byte("test submitter"))
	remote := &dto.Anchor{
		ShardId:     local.ShardId,
		Weight:      local.Weight,
		ShardSeq:    local.ShardSeq,
		ShardParent: local.ShardParent,
	}
	msg := NewShardSyncMsg(remote)
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// when peer's Anchor is same as local shard's Anchor

	// we should not have sent any ShardAncestorRequestMsg message
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when app is not registered
func TestRECV_ShardSyncMsgEvent_NoAppRegistered(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// save app's shard for later use and then unregister the app
	ShardId := stack.Anchor([]byte("test submitter")).ShardId
	stack.Unregister()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with Anchor for previously known shard, and with heavier weight so that Anchors are not same
	a := dto.TestAnchor()
	a.ShardId = ShardId
	a.Weight += 10
	msg := NewShardSyncMsg(a)
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// when peer's is known to us, even though there is no local app registered for that shard currently

	// we should have sent the ShardAncestorRequestMsg message from peer Anchor's parent to walk back on DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when remote shard is different
func TestRECV_ShardSyncMsgEvent_DifferentRemoteShard(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with default Anchor
	a := dto.TestAnchor()
	a.ShardId = []byte("some random id")
	msg := NewShardSyncMsg(a)
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// when remote shard is unknown to local sharder

	// we should have sent the ShardAncestorRequestMsg message to walk back on remote shard's DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	}
}

// stack controller listner generates RECV_NewTxBlockMsg event for unseen network message
func TestPeerListnerGeneratesEventForUnseenTxBlockMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a signed transaction followed by clean shutdown
	tx := TestSignedTransaction("test payload")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	seenNewTxBlockMsgEvent := false
	finished := make(chan struct{}, 2)
	go func() {
		tick := time.Tick(100 * time.Millisecond)
		for {
			select {
			case e := <-events:
				if e.code == RECV_NewTxBlockMsg {
					seenNewTxBlockMsgEvent = true
				} else if e.code == SHUTDOWN {
					finished <- struct{}{}
					return
				}
			case <-tick:
				finished <- struct{}{}
				return
			}
		}
	}()

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	<-finished

	// check if listener generate correct event
	if !seenNewTxBlockMsgEvent {
		t.Errorf("Event listener did not generate RECV_NewTxBlockMsg event!!!")
	}

	// we should have marked the message as seen for stack
	if !stack.isSeen(tx.Id()) {
		t.Errorf("Listener did not mark the transaction as seen while headless")
	}
}

// stack controller listner generates RECV_ShardSyncMsg event for ShardSyncMsg message
func TestPeerListnerGeneratesEventForShardSyncMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a signed transaction followed by clean shutdown
	msg := NewShardSyncMsg(&dto.Anchor{})
	mockConn.NextMsg(ShardSyncMsgCode, msg)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	seenShardSyncMsgEvent := false
	finished := make(chan struct{}, 2)
	go func() {
		tick := time.Tick(100 * time.Millisecond)
		for {
			select {
			case e := <-events:
				if e.code == RECV_ShardSyncMsg {
					seenShardSyncMsgEvent = true
				} else if e.code == SHUTDOWN {
					finished <- struct{}{}
					return
				}
			case <-tick:
				finished <- struct{}{}
				return
			}
		}
	}()

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	<-finished

	// check if listener generate correct event
	if !seenShardSyncMsgEvent {
		t.Errorf("Event listener did not generate RECV_ShardSyncMsg event!!!")
	}
}

// stack controller listner does not generate RECV_NewTxBlockMsg event for seen network message
func TestPeerListnerDoesNotGeneratesEventForSeenTxBlockMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a signed transaction followed by clean shutdown
	tx := TestSignedTransaction("test payload")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// mark the message seen with stack
	stack.isSeen(tx.Id())

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	seenNewTxBlockMsgEvent := false
	finished := make(chan struct{}, 2)
	go func() {
		tick := time.Tick(100 * time.Millisecond)
		for {
			select {
			case e := <-events:
				if e.code == RECV_NewTxBlockMsg {
					seenNewTxBlockMsgEvent = true
				} else if e.code == SHUTDOWN {
					finished <- struct{}{}
					return
				}
			case <-tick:
				finished <- struct{}{}
				return
			}
		}
	}()

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	<-finished

	// we should have marked the message as seen for stack
	if !stack.isSeen(tx.Id()) {
		t.Errorf("Listener did not mark the transaction as seen while headless")
	}

	// we should have attempted to read messaged 2 times
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should not have generated the RECV_NewTxBlockMsg event
	if seenNewTxBlockMsgEvent {
		t.Errorf("Event listener should not generate RECV_NewTxBlockMsg event for seen message!!!")
	}
}

// stack controller listner generates RECV_ShardAncestorRequestMsg event for ShardAncestorRequestMsg message
func TestPeerListnerGeneratesEventForShardAncestorRequestMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a ShardAncestorRequestMsg followed by clean shutdown
	mockConn.NextMsg(ShardAncestorRequestMsgCode, &ShardAncestorRequestMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	seenMsgEvent := false
	finished := make(chan struct{}, 2)
	go func() {
		tick := time.Tick(100 * time.Millisecond)
		for {
			select {
			case e := <-events:
				if e.code == RECV_ShardAncestorRequestMsg {
					seenMsgEvent = true
				} else if e.code == SHUTDOWN {
					finished <- struct{}{}
					return
				}
			case <-tick:
				finished <- struct{}{}
				return
			}
		}
	}()

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	<-finished

	// check if listener generate correct event
	if !seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_ShardAncestorRequestMsgCode event!!!")
	}
}

// test stack controller event listener handles RECV_ShardAncestorRequestMsgCode when start hash is known
func TestRECV_ShardAncestorRequestMsgCode_KnownStartHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

	// save the genesis hash
	gen := stack.Anchor([]byte("test submitter")).ShardParent

	// submit 2 transactions to add ancestors to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)
	// the second transaction would be Anchor's parent, and hence will be the starting hash
	tx2 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload2")
	stack.Submit(tx2)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an ancestor request message for stack's shard
	a := stack.Anchor([]byte("test submitter"))
	msg := &ShardAncestorRequestMsg{
		StartHash:    a.ShardParent,
		MaxAncestors: 10,
	}
	// now emit RECV_ShardAncestorRequestMsgCode event
	events <- newControllerEvent(RECV_ShardAncestorRequestMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard's ancestors request
	// when shard id is known and ancestor hash is valid

	// we should have fetched ancestors from sharder
	if !sharder.AncestorsCalled {
		t.Errorf("did not fetch ancestors from sharder")
	}

	// we should have sent the ShardAncestorResponseMsg message back with max ancestors
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardAncestorResponseMsg).Ancestors[0] != tx1.Id() {
		t.Errorf("Incorrect 1st ancestor: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorResponseMsg).Ancestors[0], tx1.Id())
	} else if peer.SendMsg.(*ShardAncestorResponseMsg).Ancestors[1] != gen {
		t.Errorf("Incorrect 2nd ancestor: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorResponseMsg).Ancestors[1], gen)
	}
}

// test that DLT stack does not forward a transaction that is
// rejected by application's transaction handler
func TestAppCallbackTxRejected(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, p2pLayer := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// unregister default app
	stack.Unregister()

	// define a new tx handler call back for app to always reject
	txHandlerCalled := false
	txHandler := func(tx dto.Transaction) error {
		// we reject all transactions
		txHandlerCalled = true
		return errors.New("trust no one")
	}

	// register app
	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_NewTxBlockMsg event
	tx := TestSignedTransaction("test payload")
	events <- newControllerEvent(RECV_NewTxBlockMsg, tx)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle new transaction
	if !txHandlerCalled {
		t.Errorf("Registered app's transaction handler not called")
	}

	// we should not have broadcasted message
	if p2pLayer.DidBroadcast {
		t.Errorf("Listener frowarded an invalid network transaction")
	}
}

// DLT stack controller's runner with a registered app, happy path
// (transaction message, shutdown)
func TestStackRunner(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// unregister default app
	stack.Unregister()

	// define a new tx handler call back for app that remembers when called
	gotCallback := false
	txHandler := func(tx dto.Transaction) error { gotCallback = true; return nil }

	// register app
	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// setup mock connection to send following messages:
	//    test transaction message
	//    node shutdown message
	tx, _ := shard.SignedShardTransaction("test data")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// now simulate a new connection session
	if err := stack.runner(peer); err != nil {
		t.Errorf("Peer connection has error: %s", err)
	}

	// wait for go subroutines to complete
	time.Sleep(1000 * time.Millisecond)

	// all messages should have been consumed from peer
	if mockConn.ReadCount != 2 {
		t.Errorf("Did not expect %d messages consumed from peer", mockConn.ReadCount)
	}

	// handshake message should have been sent to peer
	if mockConn.WriteCount != 1 {
		t.Errorf("Did not expect %d messages sent to peer", mockConn.WriteCount)
	}

	// verify that endorser gets called for network message
	if !endorser.TxHandlerCalled {
		t.Errorf("Endorser did not get called for network transaction")
	}
	if endorser.TxId != tx.Id() {
		t.Errorf("Endorser transaction does not match network transaction")
	}

	// application should have got call back to process transaction via sharding layer
	if !sharder.TxHandlerCalled || !gotCallback {
		t.Errorf("Application did not recieve callback via sharding layer")
	}
}
