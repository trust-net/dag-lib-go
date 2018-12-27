package stack

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"testing"
	"time"
)

func initMocks() (*dlt, *mockSharder, *mockEndorser, *p2p.MockP2P) {
	// supress all logs
	log.SetLogLevel(log.NONE)

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

	// reset the mocks to remove any state updated during initialization
	sharder.Reset()
	endorser.Reset()
	mockP2PLayer.Reset()

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
	stack, sharder, endorser, p2pLayer := initMocks()

	// register a transaction with sharder to be replayed upon app registration
	tx, _ := shard.SignedShardTransaction("test payload")
	endorser.Handle(tx)
	sharder.Handle(tx)

	// reset mocks to start tracking what we expect
	sharder.Reset()
	endorser.Reset()
	p2pLayer.Reset()

	// unregister default app and register a new app that remembers when replay transaction
	stack.Unregister()
	app := TestAppConfig()
	cbCalled := false
	txHandler := func(tx dto.Transaction) error { cbCalled = true; return nil }
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

	// we should have fetched Anchor from sharder for the app to initiate sync
	if !sharder.AnchorCalled {
		t.Errorf("Handshake did not fetch Anchor from sharding layer")
	}

	// we should have got anchor from endorsing layer (Q: Why? A: To sign it)
	if !endorser.AnchorCalled {
		t.Errorf("Handshake did not fetch Anchor from endorser layer")
	}

	// we should have broadcast the ForceShardSyncMsg to all connected peers
	if !p2pLayer.DidBroadcast {
		t.Errorf("stack did not broadcast any message")
	} else if p2pLayer.BroadcastCode != ForceShardSyncMsgCode {
		t.Errorf("Incorrect message code send: %d", p2pLayer.BroadcastCode)
	} else if string(p2pLayer.BroadcastMsg.(*ForceShardSyncMsg).Anchor.ShardId) != string(stack.app.ShardId) {
		t.Errorf("Incorrect sync for shard: %x\nExpected: %x", p2pLayer.BroadcastMsg.(*ForceShardSyncMsg).Anchor.ShardId, stack.app.ShardId)
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

	// reset mocks to start tracking what we expect
	sharder.Reset()
	endorser.Reset()

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
	stack, sharder, endorser, p2pLayer := initMocks()

	// reset mocks to start tracking what we expect
	sharder.Reset()
	endorser.Reset()
	p2pLayer.Reset()

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

	// we should not have fetched Anchor from sharder for the app to initiate sync
	if sharder.AnchorCalled {
		t.Errorf("failed registration should not fetch Anchor from sharding layer")
	}

	// we should not have got anchor from endorsing layer (Q: Why? A: To sign it)
	if endorser.AnchorCalled {
		t.Errorf("failed registration should not fetch Anchor from endorser layer")
	}

	// we should not have broadcast the ForceShardSyncMsg to all connected peers
	if p2pLayer.DidBroadcast {
		t.Errorf("stack did not broadcast any message for failed registration")
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
	//	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	stack, _, _, p2pLayer := initMocks()

	p2pLayer.Reset()

	tx := TestTransaction()
	tx.Anchor().ShardId = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for missing shard Id")
	}
	if p2pLayer.DidBroadcast {
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
	stack, sharder, endorser, p2p := initMocks()

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

	if !p2p.IsAnchored {
		t.Errorf("DLT stack did not called p2p layer's Anchor")
	}
}

// get an anchor from DLT stack when app is not registered
func TestAnchorUnregisteredApp(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, _ := initMocks()

	sharder.Reset()
	endorser.Reset()

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

	sharder.Reset()
	endorser.Reset()

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
func TestRECV_NewTxBlockMsgEvent(t *testing.T) {
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

	// we should have set the peer state
	if data := peer.GetState(int(RECV_ShardAncestorResponseMsg)); data == nil {
		t.Errorf("controller did not save last hash for ancestor response message")
	} else if state, ok := data.([64]byte); !ok {
		t.Errorf("controller saved incorrect state type: %T", data)
	} else if state != a.ShardParent {
		t.Errorf("controller saved incorrect start hash:\n%x\nExpected:\n%x", state, a.ShardParent)
	}

	// we should have sent the ShardAncestorRequestMsg message from peer Anchor's parent to walk back on DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when local shard is heavy
func TestRECV_ShardSyncMsgEvent_LessWeight(t *testing.T) {
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

	// we should set the peer state to NOT expect any ancestors response
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state != nil {
		t.Errorf("controller set expected state to incorrect hash:\n%x", state)
	}

	// we should not have sent the ShardAncestorRequestMsg message
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}
}

// test stack controller event listener handles RECV_ShardSyncMsg correctly when both weights are same
func TestRECV_ShardSyncMsgEvent_SameWeight_NumericHeavy(t *testing.T) {
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

	// we should set the peer state to expect ancestors response for requested hash
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state == nil || state.([64]byte) != a.ShardParent {
		t.Errorf("controller set expected state to incorrect hash:\n%x\nExpected:\n%x", state, a.ShardParent)
	}

	// we should have sent the ShardAncestorRequestMsg message from peer Anchor's parent to walk back on DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	}
}

func TestRECV_ShardSyncMsgEvent_LessWeight_NumericHeavy(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// set some random value in peer's state, to validate later
	peer.SetState(int(RECV_ShardAncestorResponseMsg), dto.RandomHash())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build a shard sync message with less weight but parent hash that is higher numeric value
	a := stack.Anchor([]byte("test submitter"))
	a.Weight -= 1
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

	// we should set the peer state to NOT expect any ancestors response
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state != nil {
		t.Errorf("controller set expected state to incorrect hash:\n%x", state)
	}

	// we should not have sent the ShardAncestorRequestMsg message from peer Anchor's parent to walk back on DAG
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
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

	// set some random value in peer's state, to validate later
	peer.SetState(int(RECV_ShardAncestorResponseMsg), dto.RandomHash())

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

	// we should set the peer state to NOT expect any ancestors response
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state != nil {
		t.Errorf("controller set expected state to incorrect hash:\n%x", state)
	}

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

	// we should set the peer state to expect ancestors response for requested hash
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state == nil || state.([64]byte) != a.ShardParent {
		t.Errorf("controller set expected state to incorrect hash:\n%x\nExpected:\n%x", state, a.ShardParent)
	}

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

	// we should set the peer state to expect ancestors response for requested hash
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state == nil || state.([64]byte) != a.ShardParent {
		t.Errorf("controller set expected state to incorrect hash:\n%x\nExpected:\n%x", state, a.ShardParent)
	}

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
	finished := checkForEventCode(RECV_NewTxBlockMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
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
	finished := checkForEventCode(RECV_ShardSyncMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
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
	finished := checkForEventCode(RECV_NewTxBlockMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// we should have marked the message as seen for stack
	if !stack.isSeen(tx.Id()) {
		t.Errorf("Listener did not mark the transaction as seen while headless")
	}

	// we should have attempted to read messaged 2 times
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should not have generated the RECV_NewTxBlockMsg event
	if result.seenMsgEvent {
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
	finished := checkForEventCode(RECV_ShardAncestorRequestMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
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

// stack controller listner generates RECV_ShardAncestorResponseMsg event for ShardAncestorResponseMsg message
func TestPeerListnerGeneratesEventForShardAncestorResponseMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a ShardAncestorResponseMsg followed by clean shutdown
	mockConn.NextMsg(ShardAncestorResponseMsgCode, &ShardAncestorResponseMsg{})
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
				if e.code == RECV_ShardAncestorResponseMsg {
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
		t.Errorf("Event listener did not generate RECV_ShardAncestorResponseMsg event!!!")
	}
}

// test stack controller event listener handles RECV_ShardAncestorResponseMsg when all hashesh are unknown
func TestRECV_ShardAncestorResponseMsg_AllUnknown(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the start hash with peer
	startHash := dto.RandomHash()
	peer.SetState(int(RECV_ShardAncestorResponseMsg), startHash)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an ancestor response message with hashes of unknown transactions
	msg := &ShardAncestorResponseMsg{
		StartHash: startHash,
	}
	for i := 0; i < 10; i++ {
		msg.Ancestors = append(msg.Ancestors, dto.RandomHash())
	}
	// now emit RECV_ShardAncestorResponseMsg event
	events <- newControllerEvent(RECV_ShardAncestorResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard's ancestors response
	// when all hashes are unknown

	// we should have fetched the peer state to validate ancestor response's starting hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should have set the peer state to last of the unknown ancestors
	if data := peer.GetState(int(RECV_ShardAncestorResponseMsg)); data == nil {
		t.Errorf("controller did not save last hash for ancestor response message")
	} else if state, ok := data.([64]byte); !ok {
		t.Errorf("controller saved incorrect state type: %T", data)
	} else if state != msg.Ancestors[9] {
		t.Errorf("controller saved incorrect start hash:\n%x\nExpected:\n%x", state, msg.Ancestors[9])
	}

	// we should have sent the ShardAncestorRequestMsg message back to fetch more ancestors
	// starting from top ancestor in ShardAncestorResponseMsg
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardAncestorRequestMsg).StartHash != msg.Ancestors[9] {
		t.Errorf("Incorrect start hash: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorRequestMsg).StartHash, msg.Ancestors[9])
	}
}

// test stack controller event listener handles RECV_ShardAncestorResponseMsg when last hash state is not correct
func TestRECV_ShardAncestorResponseMsg_IncorrectState(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the start hash with peer
	startHash := dto.RandomHash()
	peer.SetState(int(RECV_ShardAncestorResponseMsg), startHash)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an ancestor response message with incorrect start hash
	msg := &ShardAncestorResponseMsg{
		StartHash: dto.RandomHash(),
	}
	for i := 0; i < 10; i++ {
		msg.Ancestors = append(msg.Ancestors, dto.RandomHash())
	}
	// now emit RECV_ShardAncestorResponseMsg event
	events <- newControllerEvent(RECV_ShardAncestorResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard's ancestors response
	// when all hashes are unknown

	// we should have fetched the peer state to validate ancestor response's starting hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should not have changed the peer state
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)).([64]byte); state != startHash {
		t.Errorf("controller changed start hash for incorrect state:\n%x\nExpected:\n%x", state, startHash)
	}

	// we should not have sent the ShardAncestorRequestMsg message back to fetch more ancestors
	// starting from top ancestor in ShardAncestorResponseMsg
	if peer.SendCalled {
		t.Errorf("should not send any message to peer for incorrect state")
	}
}

// test stack controller event listener handles RECV_ShardAncestorResponseMsg there is a known common ancestor
func TestRECV_ShardAncestorResponseMsg_KnownAncestor(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the start hash with peer
	startHash := dto.RandomHash()
	peer.SetState(int(RECV_ShardAncestorResponseMsg), startHash)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an ancestor response message with common known hash as 6th item
	msg := &ShardAncestorResponseMsg{
		StartHash: startHash,
	}
	for i := 0; i < 5; i++ {
		msg.Ancestors = append(msg.Ancestors, dto.RandomHash())
	}
	msg.Ancestors = append(msg.Ancestors, tx.Id())

	// now emit RECV_ShardAncestorResponseMsg event
	events <- newControllerEvent(RECV_ShardAncestorResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard's ancestors response
	// when all hashes are unknown

	// we should have fetched the peer state to validate ancestor response's starting hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should have set the peer state to last known common ancestor for RECV_ShardAncestorResponseMsg state
	// (this is to ensure that any DoS gets blocked right away)
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)).([64]byte); state != tx.Id() {
		t.Errorf("controller did not save start hash to common ancestor:\n%x\nExpected:\n%x", state, tx.Id())
	}

	// we should also have set the peer state to last known common ancestor for RECV_ShardChildrenResponseMsg state
	// (this is to ensure that any DoS gets blocked right away)
	if data := peer.GetState(int(RECV_ShardChildrenResponseMsg)); data == nil {
		t.Errorf("controller did not save last hash for children response message")
	} else if state, ok := data.([64]byte); !ok {
		t.Errorf("controller saved incorrect state type: %T", data)
	} else if state != tx.Id() {
		t.Errorf("controller saved incorrect parent hash:\n%x\nExpected:\n%x", state, tx.Id())
	}

	// we should have sent the ShardChildrenRequestMsg message to start populating the DAG from peer's shard
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardChildrenRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardChildrenRequestMsg).Parent != tx.Id() {
		t.Errorf("Incorrect start hash: %x\nExpected: %x", peer.SendMsg.(*ShardChildrenRequestMsg).Parent, tx.Id())
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

	// reset p2pLayer, since new registration would have caused broadcast
	p2pLayer.Reset()

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

// stack controller listner generates RECV_ShardChildrenRequestMsg event for ShardChildrenRequestMsg message
func TestPeerListnerGeneratesEventForShardChildrenRequestMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a ShardAncestorRequestMsg followed by clean shutdown
	mockConn.NextMsg(ShardChildrenRequestMsgCode, &ShardChildrenRequestMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_ShardChildrenRequestMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_ShardChildrenRequestMsg event!!!")
	}
}

// test stack controller event listener handles RECV_ShardChildrenRequestMsg when hash is known
func TestRECV_ShardChildrenRequestMsg_KnownHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

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

	// build an children request message for stack's shard
	msg := &ShardChildrenRequestMsg{
		Parent: tx1.Id(),
	}
	// now emit RECV_ShardChildrenRequestMsg event
	events <- newControllerEvent(RECV_ShardChildrenRequestMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children request
	// when hash is known and children are present

	// we should have fetched children from sharder
	if !sharder.ChildrenCalled {
		t.Errorf("did not fetch children from sharder")
	}

	// we should have sent the ShardChildrenResponseMsg message back with max ancestors
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardChildrenResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardChildrenResponseMsg).Parent != tx1.Id() {
		t.Errorf("Incorrect parent hash: %x\nExpected: %x", peer.SendMsg.(*ShardChildrenResponseMsg).Parent, tx1.Id())
	} else if len(peer.SendMsg.(*ShardChildrenResponseMsg).Children) != 1 {
		t.Errorf("Incorrect number of children: %d, Expected: %d", len(peer.SendMsg.(*ShardChildrenResponseMsg).Children), 1)
	} else if peer.SendMsg.(*ShardChildrenResponseMsg).Children[0] != tx2.Id() {
		t.Errorf("Incorrect 1st child: %x\nExpected: %x", peer.SendMsg.(*ShardChildrenResponseMsg).Children[0], tx2.Id())
	}
}

// test stack controller event listener handles RECV_ShardChildrenRequestMsg when hash is unknown
func TestRECV_ShardChildrenRequestMsg_UnknownHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

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

	// build an children request message for stack's shard with unknown hash
	msg := &ShardChildrenRequestMsg{
		Parent: dto.RandomHash(),
	}
	// now emit RECV_ShardChildrenRequestMsg event
	events <- newControllerEvent(RECV_ShardChildrenRequestMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children request
	// when hash is unknown

	// we should have fetched children from sharder
	if !sharder.ChildrenCalled {
		t.Errorf("did not fetch children from sharder")
	}

	// we should have sent the ShardChildrenResponseMsg message back with 0 children
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardChildrenResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardChildrenResponseMsg).Parent != msg.Parent {
		t.Errorf("Incorrect parent hash: %x\nExpected: %x", peer.SendMsg.(*ShardChildrenResponseMsg).Parent, msg.Parent)
	} else if len(peer.SendMsg.(*ShardChildrenResponseMsg).Children) != 0 {
		t.Errorf("Incorrect number of children: %d, Expected: %d", len(peer.SendMsg.(*ShardChildrenResponseMsg).Children), 0)
	}
}

func checkForEventCode(code eventEnum, events chan controllerEvent) chan struct{ seenMsgEvent bool } {
	finished := make(chan struct{ seenMsgEvent bool }, 2)
	go func() {
		tick := time.Tick(100 * time.Millisecond)
		for {
			select {
			case e := <-events:
				if e.code == code {
					finished <- struct{ seenMsgEvent bool }{seenMsgEvent: true}
				} else if e.code == SHUTDOWN {
					finished <- struct{ seenMsgEvent bool }{}
					return
				}
			case <-tick:
				finished <- struct{ seenMsgEvent bool }{}
				return
			}
		}
	}()
	return finished
}

// stack controller listner generates RECV_ShardChildrenResponseMsg event for ShardChildrenResponseMsg message
func TestPeerListnerGeneratesEventForShardChildrenResponseMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a ShardAncestorRequestMsg followed by clean shutdown
	mockConn.NextMsg(ShardChildrenResponseMsgCode, &ShardChildrenResponseMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_ShardChildrenResponseMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_ShardChildrenResponseMsg event!!!")
	}
}

// test stack controller event listener handles RECV_ShardChildrenResponseMsg when Parent hash is unexpected
func TestRECV_ShardChildrenResponseMsg_UnexpectedHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the start hash with peer
	peer.SetState(int(RECV_ShardChildrenResponseMsg), tx1.Id())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an children response message using parent hash known to shard
	msg := &ShardChildrenResponseMsg{
		Parent:   dto.RandomHash(),
		Children: [][64]byte{},
	}
	for i := 0; i < 5; i++ {
		msg.Children = append(msg.Children, dto.RandomHash())
	}

	// now emit RECV_ShardChildrenResponseMsg event
	events <- newControllerEvent(RECV_ShardChildrenResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children response message
	// when parent hash of children does not match expected state

	// we should have fetched the peer state to validate ancestor response's starting hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should not have changed the peer state
	if state := peer.GetState(int(RECV_ShardChildrenResponseMsg)).([64]byte); state != tx1.Id() {
		t.Errorf("controller changed start hash for incorrect state:\n%x\nExpected:\n%x", state, tx1.Id())
	}

	// we should not have fetched queue to get or set
	if peer.ShardChildrenQCallCount != 0 {
		t.Errorf("should not access peer's shard children queue")
	}

	// we should not have emit the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 0 {
		t.Errorf("should not emit any event for incorrect state")
	}
}

// test stack controller event listener handles RECV_ShardChildrenResponseMsg when Parent hash is as expected
func TestRECV_ShardChildrenResponseMsg_ExpectedHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the start hash with peer
	peer.SetState(int(RECV_ShardChildrenResponseMsg), tx1.Id())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an children response message using parent hash known to shard
	msg := &ShardChildrenResponseMsg{
		Parent:   tx1.Id(),
		Children: [][64]byte{},
	}
	for i := 0; i < 5; i++ {
		msg.Children = append(msg.Children, dto.RandomHash())
	}

	// now emit RECV_ShardChildrenResponseMsg event
	events <- newControllerEvent(RECV_ShardChildrenResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children response message
	// when parent hash of children does not match expected state

	// we should have fetched the peer state to validate ancestor response's starting hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should have changed the peer state to null so that no further children response can be processed
	if state := peer.GetState(int(RECV_ShardChildrenResponseMsg)).([64]byte); state != [64]byte{} {
		t.Errorf("controller did not change start hash, current:\n%x\nExpected:\n%x", state, [64]byte{})
	}

	// we should have fetched queue to add children for processing
	if peer.ShardChildrenQCallCount == 0 {
		t.Errorf("did not access peer's shard children queue")
	}

	// we should have added all children to shard children q
	if peer.ShardChildrenQ().Count() != 5 {
		t.Errorf("incorrect number of children in queue: %d, expected: %d", peer.ShardChildrenQ().Count(), 5)
	}

	// we should have emitted the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 1 {
		t.Errorf("did not emit event for child processing")
	}
}

// test stack controller event listener handles RECV_ShardChildrenResponseMsg when one of the children is already known
func TestRECV_ShardChildrenResponseMsg_KnownChild(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// submit another transactions to add child to local shard's Anchor
	tx2 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload2")
	stack.Submit(tx2)

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the start hash with peer
	peer.SetState(int(RECV_ShardChildrenResponseMsg), tx1.Id())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an children response message using parent hash known to shard
	msg := &ShardChildrenResponseMsg{
		Parent:   tx1.Id(),
		Children: [][64]byte{},
	}
	for i := 0; i < 4; i++ {
		msg.Children = append(msg.Children, dto.RandomHash())
	}

	// make one of the child known to local shard
	msg.Children = append(msg.Children, tx2.Id())

	// now emit RECV_ShardChildrenResponseMsg event
	events <- newControllerEvent(RECV_ShardChildrenResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children response message
	// when parent hash of children does not match expected state

	// we should have fetched the peer state to validate ancestor response's starting hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should have changed the peer state to null so that no further children response can be processed
	if state := peer.GetState(int(RECV_ShardChildrenResponseMsg)).([64]byte); state != [64]byte{} {
		t.Errorf("controller did not change start hash, current:\n%x\nExpected:\n%x", state, [64]byte{})
	}

	// we should have fetched queue to add children for processing
	if peer.ShardChildrenQCallCount == 0 {
		t.Errorf("did not access peer's shard children queue")
	}

	// we should have added all but 1 children to shard children q
	if peer.ShardChildrenQ().Count() != 4 {
		t.Errorf("incorrect number of children in queue: %d, expected: %d", peer.ShardChildrenQ().Count(), 4)
	}

	// we should have emitted the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 1 {
		t.Errorf("did not emit event for child processing")
	}
}

// test stack controller event listener handles POP_ShardChild when there is a child in shard children queue
func TestPOP_ShardChild_HasChild(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// add 2 children to peer's shard children queue
	child1 := dto.RandomHash()
	child2 := dto.RandomHash()
	peer.ShardChildrenQ().Push(child1)
	peer.ShardChildrenQ().Push(child2)
	peer.Reset()

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit POP_ShardChild event
	events <- newControllerEvent(POP_ShardChild, nil)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle pop shard child event
	// when shard children queue has children

	// we should have fetched queue to pop child
	if peer.ShardChildrenQCallCount == 0 {
		t.Errorf("did not access peer's shard children queue")
	}

	// peer's shard children queue should have 1 less element left
	if peer.ShardChildrenQ().Count() != 1 {
		t.Errorf("incorrect number of children in queue: %d", peer.ShardChildrenQ().Count())
	}

	// we should not have emit the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 0 {
		t.Errorf("should not emit any event until TxBlockShardResponseMsg is recieved")
	}

	// we should set the peer state to expect shard child transaction for requested hash
	if state := peer.GetState(int(RECV_TxShardChildResponseMsg)); state == nil || state.([64]byte) != child1 {
		t.Errorf("controller set expected state to incorrect hash:\n%x\nExpected:\n%x", state, child1)
	}

	// we should have sent TxBlockShardRequestMsg to peer, to get child transaction and its descendents
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != TxShardChildRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*TxShardChildRequestMsg).Hash != child1 {
		t.Errorf("Incorrect child tx requested: %x\nExpected: %x", peer.SendMsg.(*TxShardChildRequestMsg).Hash, child1)
	}
}

// test stack controller event listener handles POP_ShardChild when there is no child in shard children queue
func TestPOP_ShardChild_NoChild(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer with empty queue
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit POP_ShardChild event
	events <- newControllerEvent(POP_ShardChild, nil)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle pop shard child event
	// when shard children queue has no children

	// we should not set the peer state to expect shard child transaction
	if state := peer.GetState(int(RECV_TxShardChildResponseMsg)); state != nil {
		t.Errorf("controller set expected state when it should not")
	}

	// we should have fetched queue to pop child
	if peer.ShardChildrenQCallCount == 0 {
		t.Errorf("did not access peer's shard children queue")
	}

	// we should not have sent TxBlockShardRequestMsg to peer, to get child transaction and its descendents
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}
}

// stack controller listner generates RECV_TxShardChildRequestMsg event for TxShardChildRequestMsg message
func TestPeerListnerGeneratesEventForTxShardChildRequestMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a TxShardChildRequestMsg followed by clean shutdown
	mockConn.NextMsg(TxShardChildRequestMsgCode, &TxShardChildRequestMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_TxShardChildRequestMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_TxShardChildRequestMsg event!!!")
	}
}

// test stack controller event listener handles RECV_TxShardChildRequestMsg when hash is known
func TestRECV_TxShardChildRequestMsg_KnownHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

	// submit 2 transactions to add a tx and its child to local shard's Anchor
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

	// build a transaction request with shard children for known hash
	msg := &TxShardChildRequestMsg{
		Hash: tx1.Id(),
	}
	// now emit RECV_TxShardChildRequestMsg event
	events <- newControllerEvent(RECV_TxShardChildRequestMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children request
	// when hash is known and children are present

	// we should have fetched children from sharder
	if !sharder.ChildrenCalled {
		t.Errorf("did not fetch children from sharder")
	}

	// we should have sent the TxShardChildResponseMsg message back with 1 child
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != TxShardChildResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if tx1Bytes, _ := tx1.Serialize(); string(peer.SendMsg.(*TxShardChildResponseMsg).Bytes) != string(tx1Bytes) {
		t.Errorf("Incorrect transaction hash: %x\nExpected: %x", peer.SendMsg.(*TxShardChildResponseMsg).Bytes, tx1Bytes)
	} else if len(peer.SendMsg.(*TxShardChildResponseMsg).Children) != 1 {
		t.Errorf("Incorrect number of children: %d, Expected: %d", len(peer.SendMsg.(*TxShardChildResponseMsg).Children), 1)
	} else if peer.SendMsg.(*TxShardChildResponseMsg).Children[0] != tx2.Id() {
		t.Errorf("Incorrect 1st child: %x\nExpected: %x", peer.SendMsg.(*TxShardChildResponseMsg).Children[0], tx2.Id())
	}
}

// test stack controller event listener handles RECV_TxShardChildRequestMsg when hash is unknown
func TestRECV_TxShardChildRequestMsg_UnknownHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, _, _ := initMocks()

	// submit 2 transactions to add a tx and its child to local shard's Anchor
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

	// build a transaction request with shard children for unknown hash
	msg := &TxShardChildRequestMsg{
		Hash: dto.RandomHash(),
	}
	// now emit RECV_TxShardChildRequestMsg event
	events <- newControllerEvent(RECV_TxShardChildRequestMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle transaction request
	// when hash is unknown

	// we should not have fetched children from sharder
	if sharder.ChildrenCalled {
		t.Errorf("should not fetch children from sharder")
	}

	// we should not have sent the TxShardChildResponseMsg message
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}

	// because its an error condition, we should have disconnected
	if !peer.DisconnectCalled {
		t.Errorf("did not disconnect with peer")
	}
}

// stack controller listner generates RECV_TxShardChildResponseMsg event for TxShardChildResponseMsg message
func TestPeerListnerGeneratesEventForTxShardChildResponseMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a TxShardChildResponseMsg followed by clean shutdown
	mockConn.NextMsg(TxShardChildResponseMsgCode, &TxShardChildResponseMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_TxShardChildResponseMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_TxShardChildResponseMsg event!!!")
	}
}

// test stack controller event listener handles RECV_TxShardChildResponseMsg when transaction hash is unexpected
func TestRECV_TxShardChildResponseMsg_UnexpectedHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()

	// build another transaction as child of above transaction to send in response
	tx2 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the expected hash with peer's state
	peer.SetState(int(RECV_TxShardChildResponseMsg), tx2.Id())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an transaction response message using hash unknown to shard
	msg := NewTxShardChildResponseMsg(dto.TestSignedTransaction("test data"), [][64]byte{})
	for i := 0; i < 5; i++ {
		msg.Children = append(msg.Children, dto.RandomHash())
	}

	// now emit RECV_TxShardChildResponseMsg event
	events <- newControllerEvent(RECV_TxShardChildResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children response message
	// when parent hash of children does not match expected state

	// we should have fetched the peer state to validate transaction response's hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should not have changed the peer state
	if state := peer.GetState(int(RECV_TxShardChildResponseMsg)); state == nil || state.([64]byte) != tx2.Id() {
		t.Errorf("controller changed start hash for incorrect state:\n%x\nExpected:\n%x", state, tx2.Id())
	}

	// we should not have marked the message as seen for stack
	if stack.isSeen(tx2.Id()) {
		t.Errorf("stack should not mark the unexpected transaction as seen")
	}

	// we should not have broadcasted message
	if p2pLayer.DidBroadcast {
		t.Errorf("stack should not froward unexpected transaction")
	}

	// sharding layer should not be asked to handle transaction
	if sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller should not call sharding layer for unexpected transaction")
	}

	// verify that endorser does not gets called for un-serializable message
	if endorser.TxHandlerCalled {
		t.Errorf("Endorser should not get called for unexpected transaction")
	}

	// we should not have added children of unexpected transaction to shard children queue
	if peer.ShardChildrenQ().Count() != 0 {
		t.Errorf("incorrect number of children in queue: %d, expected: %d", peer.ShardChildrenQ().Count(), 0)
	}

	// we should not have emitted the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 0 {
		t.Errorf("should not emit event for next child processing")
	}
}

// test stack controller event listener handles RECV_TxShardChildResponseMsg when transaction is corrupted
func TestRECV_TxShardChildResponseMsg_UnserializableTx(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()

	// build another transaction as child of above transaction to send in response
	tx2 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the expected hash with peer's state
	peer.SetState(int(RECV_TxShardChildResponseMsg), tx2.Id())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an transaction response message using hash unknown to shard
	msg := &TxShardChildResponseMsg{
		Bytes:    []byte("some corrupted bytes"),
		Children: [][64]byte{},
	}
	for i := 0; i < 5; i++ {
		msg.Children = append(msg.Children, dto.RandomHash())
	}

	// now emit RECV_TxShardChildResponseMsg event
	events <- newControllerEvent(RECV_TxShardChildResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children response message
	// when parent hash of children does not match expected state

	// we should not event have fetched the peer state to validate transaction response's hash
	if peer.GetStateCalled {
		t.Errorf("should not get state from peer to for unserializable transaction")
	}

	// we should not have changed the peer state
	if state := peer.GetState(int(RECV_TxShardChildResponseMsg)); state == nil || state.([64]byte) != tx2.Id() {
		t.Errorf("controller changed start hash for incorrect state:\n%x\nExpected:\n%x", state, tx2.Id())
	}

	// we should not have marked the message as seen for stack
	if stack.isSeen(tx2.Id()) {
		t.Errorf("stack should not mark the un-serializable transaction as seen")
	}

	// we should not have broadcasted message
	if p2pLayer.DidBroadcast {
		t.Errorf("stack should not froward un-serializable transaction")
	}

	// sharding layer should not be asked to handle transaction
	if sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller should not call sharding layer for un-serializable transaction")
	}

	// verify that endorser does not gets called for un-serializable message
	if endorser.TxHandlerCalled {
		t.Errorf("Endorser should not get called for un-serializable transaction")
	}

	// we should not have added children of un-serializable transaction to shard children queue
	if peer.ShardChildrenQ().Count() != 0 {
		t.Errorf("incorrect number of children in queue: %d, expected: %d", peer.ShardChildrenQ().Count(), 0)
	}

	// we should not have emitted the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 0 {
		t.Errorf("should not emit event for next child processing")
	}
}

// test stack controller event listener handles RECV_TxShardChildResponseMsg when transaction hash is expected
func TestRECV_TxShardChildResponseMsg_ExpectedHash(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// submit a transactions to add ancestor to local shard's Anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()

	// build another transaction as child of above transaction to send in response
	tx2 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// save the expected hash with peer's state
	peer.SetState(int(RECV_TxShardChildResponseMsg), tx2.Id())

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// build an transaction response message using hash unknown to shard
	msg := NewTxShardChildResponseMsg(tx2, [][64]byte{})
	for i := 0; i < 5; i++ {
		msg.Children = append(msg.Children, dto.RandomHash())
	}

	// now emit RECV_TxShardChildResponseMsg event
	events <- newControllerEvent(RECV_TxShardChildResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle children response message
	// when parent hash of children does not match expected state

	// we should have fetched the peer state to validate transaction response's hash
	if !peer.GetStateCalled {
		t.Errorf("did not get state from peer to validate response")
	}

	// we should have changed the peer state to null
	if state := peer.GetState(int(RECV_TxShardChildResponseMsg)); state == nil || state.([64]byte) != [64]byte{} {
		t.Errorf("controller changed hash to incorrect state:\n%x\nExpected:\n%x", state, [64]byte{})
	}

	// we should have marked the message as seen for stack
	if !stack.isSeen(tx2.Id()) {
		t.Errorf("Listener did not mark the received transaction as seen")
	}

	// we should have broadcasted message
	if !p2pLayer.DidBroadcast {
		t.Errorf("Listener did not froward received transaction")
	}

	// sharding layer should be asked to handle transaction
	if !sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller did not call sharding layer")
	}

	// verify that endorser gets called for network message
	if !endorser.TxHandlerCalled {
		t.Errorf("Endorser did not get called for recieved transaction")
	}

	// we should have added children of expected transaction to shard children queue
	if peer.ShardChildrenQ().Count() != 5 {
		t.Errorf("incorrect number of children in queue: %d, expected: %d", peer.ShardChildrenQ().Count(), 5)
	}

	// we should have emitted the POP_ShardChild event to process ShardChildrenQ
	if len(events) != 1 {
		t.Errorf("did not emit event for next child processing")
	}
}

// stack controller listner generates RECV_ForceShardSyncMsg event for ForceShardSyncMsg message
func TestPeerListnerGeneratesEventForForceShardSyncMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a signed transaction followed by clean shutdown
	msg := NewForceShardSyncMsg(&dto.Anchor{})
	mockConn.NextMsg(ForceShardSyncMsgCode, msg)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_ForceShardSyncMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_ForceShardSyncMsg event!!!")
	}
}

// test stack controller event listener handles RECV_ForceShardSyncMsg when shard is known and up-to-date
func TestRECV_ForceShardSyncMsg_KnownShard_InSync(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// submit a transactions to local shard
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// save app's shard's anchor for later use and then unregister the app
	anchor := stack.Anchor([]byte("test submitter"))

	// now unregister default app, and register a different app/shard
	stack.Unregister()
	stack.Register([]byte("a different shard"), "shard-2", func(tx dto.Transaction) error { return nil })

	// reset the mocks to remove any state updated during initialization
	sharder.Reset()
	endorser.Reset()
	p2pLayer.Reset()

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

	// build a ForceShardSyncMsg request with anchor of previously known (and up-to-date) shard
	msg := NewForceShardSyncMsg(anchor)

	// now emit RECV_TxShardChildRequestMsg event
	events <- newControllerEvent(RECV_ForceShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle ForceShardSyncMsg request
	// when shard is known (either currently registered, or have some history for shard)

	// we should have fetched anchor from sharder for requested shard
	if !sharder.SyncAnchorCalled {
		t.Errorf("did not fetch sync anchor from sharder")
	}

	// we should have got anchor updated by endorsing layer
	if !endorser.AnchorCalled {
		t.Errorf("did not fetch Anchor from endorser layer")
	}

	// we should have got anchor signed by endorsing layer
	if !p2pLayer.IsAnchored {
		t.Errorf("did not sign Anchor with p2p layer")
	}

	// we should not have sent any message for in-sync shard
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}
}

// test stack controller event listener handles RECV_ForceShardSyncMsg when shard is known and local node is ahead
func TestRECV_ForceShardSyncMsg_KnownShard_LocalAhead(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// save app's shard's anchor for later use and then unregister the app
	anchor := stack.Anchor([]byte("test submitter"))

	// submit a transactions to local shard to move it ahead of saved anchor
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// now unregister default app, and register a different app/shard
	stack.Unregister()
	stack.Register([]byte("a different shard"), "shard-2", func(tx dto.Transaction) error { return nil })

	// reset the mocks to remove any state updated during initialization
	sharder.Reset()
	endorser.Reset()
	p2pLayer.Reset()

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

	// build a ForceShardSyncMsg request with anchor of previously known shard
	msg := NewForceShardSyncMsg(anchor)

	// now emit RECV_TxShardChildRequestMsg event
	events <- newControllerEvent(RECV_ForceShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle ForceShardSyncMsg request
	// when shard is known (either currently registered, or have some history for shard)

	// we should have fetched anchor from sharder for requested shard
	if !sharder.SyncAnchorCalled {
		t.Errorf("did not fetch sync anchor from sharder")
	}

	// we should have got anchor updated by endorsing layer
	if !endorser.AnchorCalled {
		t.Errorf("did not fetch Anchor from endorser layer")
	}

	// we should have got anchor signed by endorsing layer
	if !p2pLayer.IsAnchored {
		t.Errorf("did not sign Anchor with p2p layer")
	}

	// we should have sent the ShardSyncMsg message back for remote peer to start sync
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardSyncMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardSyncMsg).Anchor.ShardParent != tx1.Id() {
		t.Errorf("Incorrect ShardSyncMsg Parent: %x\nExpected: %x", peer.SendMsg.(*ShardSyncMsg).Anchor.ShardParent, tx1.Id())
	}
}

// test stack controller event listener handles RECV_ForceShardSyncMsg when shard is known and remote node is ahead
func TestRECV_ForceShardSyncMsg_KnownShard_RemoteAhead(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// submit a transactions to local shard
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// save app's shard's anchor for later use and then unregister the app
	anchor := stack.Anchor([]byte("test submitter"))

	// now unregister default app, and register a different app/shard
	stack.Unregister()
	stack.Register([]byte("a different shard"), "shard-2", func(tx dto.Transaction) error { return nil })

	// reset the mocks to remove any state updated during initialization
	sharder.Reset()
	endorser.Reset()
	p2pLayer.Reset()

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

	// build a ForceShardSyncMsg request with anchor of previously known shard with higher weight
	anchor.Weight += 1
	msg := NewForceShardSyncMsg(anchor)

	// now emit RECV_TxShardChildRequestMsg event
	events <- newControllerEvent(RECV_ForceShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle ForceShardSyncMsg request
	// when shard is known (either currently registered, or have some history for shard)

	// we should have fetched anchor from sharder for requested shard
	if !sharder.SyncAnchorCalled {
		t.Errorf("did not fetch sync anchor from sharder")
	}

	// we should have got anchor updated by endorsing layer
	if !endorser.AnchorCalled {
		t.Errorf("did not fetch Anchor from endorser layer")
	}

	// we should have got anchor signed by endorsing layer
	if !p2pLayer.IsAnchored {
		t.Errorf("did not sign Anchor with p2p layer")
	}

	// we should have sent the ShardAncestorRequestMsg message to start sync with peer
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardAncestorRequestMsg).StartHash != anchor.ShardParent {
		t.Errorf("Incorrect ShardSyncMsg Parent: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorRequestMsg).StartHash, anchor.ShardParent)
	}
}

// test stack controller event listener handles RECV_ForceShardSyncMsg when shard is unknown
func TestRECV_ForceShardSyncMsg_UnknownShard(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer := initMocks()

	// submit a transactions to local shard
	tx1 := TestAnchoredTransaction(stack.Anchor([]byte("test submitter")), "test payload1")
	stack.Submit(tx1)

	// reset the mocks to remove any state updated during initialization
	sharder.Reset()
	endorser.Reset()
	p2pLayer.Reset()

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

	// create another stack instance as peer
	peerStack, _, _, _ := initMocks()

	// now unregister default app, and register a different app/shard
	peerStack.Unregister()
	peerStack.Register([]byte("a different shard"), "shard-2", func(tx dto.Transaction) error { return nil })

	// build a ForceShardSyncMsg request with anchor of remoteStack using an unknown shard
	anchor := peerStack.Anchor([]byte("test submitter"))
	msg := NewForceShardSyncMsg(anchor)

	// now emit RECV_TxShardChildRequestMsg event
	events <- newControllerEvent(RECV_ForceShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle ForceShardSyncMsg request
	// when shard is known (either currently registered, or have some history for shard)

	// we should have fetched anchor from sharder for requested shard
	if !sharder.SyncAnchorCalled {
		t.Errorf("did not fetch sync anchor from sharder")
	}

	// we should have got anchor updated by endorsing layer
	if !endorser.AnchorCalled {
		t.Errorf("did not fetch Anchor from endorser layer")
	}

	// we should have got anchor signed by endorsing layer
	if !p2pLayer.IsAnchored {
		t.Errorf("did not sign Anchor with p2p layer")
	}

	// we should have sent the ShardAncestorRequestMsg message to start sync with peer
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardAncestorRequestMsg).StartHash != anchor.ShardParent {
		t.Errorf("Incorrect ShardSyncMsg Parent: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorRequestMsg).StartHash, anchor.ShardParent)
	}
}

// test stack controller event listener handles RECV_NewTxBlockMsg correctly when transaction's parent is unknown
func TestRECV_NewTxBlockMsg_UnknownTxParent(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	//	log.SetLogLevel(log.DEBUG)
	//	defer log.SetLogLevel(log.NONE)

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

	// build a new transaction message with unknown parent
	tx := TestSignedTransaction("test payload")
	tx.Anchor().ShardParent = dto.RandomHash()

	// now emit RECV_NewTxBlockMsg event
	events <- newControllerEvent(RECV_NewTxBlockMsg, tx)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle new transaction
	// when transaction's shard parent is unknown to local node

	// we should set the peer state to expect ancestors response for requested hash
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state == nil || state.([64]byte) != tx.Anchor().ShardParent {
		t.Errorf("controller set expected state to incorrect hash:\n%x\nExpected:\n%x", state, tx.Anchor().ShardParent)
	}

	// we should have sent the ShardAncestorRequestMsg message to initiate WalkUpStage on remote peer's shard DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardAncestorRequestMsg).StartHash != tx.Anchor().ShardParent {
		t.Errorf("Incorrect walk up hash: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorRequestMsg).StartHash, tx.Anchor().ShardParent)
	}
}
