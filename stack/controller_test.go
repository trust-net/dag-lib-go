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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	app := TestAppConfig()
	cbCalled := false
	txHandler := func(tx dto.Transaction) error { cbCalled = true; return nil }

	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser
	// register a network transaction with sharder
	tx, _ := shard.SignedShardTransaction("test payload")
	if err := stack.endorser.Handle(tx); err != nil {
		t.Errorf("Endorser network transaction failed, err: %s", err)
	}
	if err := stack.sharder.Handle(tx); err != nil {
		t.Errorf("Sharder network transaction failed, err: %s", err)
	}

	if err := stack.Register(tx.Anchor().ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return errors.New("forced failure") }

	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock endorser into stack
	stack.endorser = NewMockEndorser(stack.db)
	// register a transaction with sharder
	tx, _ := shard.SignedShardTransaction("test payload")
	stack.sharder.Handle(tx)

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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	app := AppConfig{}
	txHandler := func(tx dto.Transaction) error { return nil }

	// register app one time
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser

	// attempt to register app again
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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	txHandler := func(tx dto.Transaction) error { return nil }

	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock endorser into stack
	stack.endorser = NewMockEndorser(stack.db)

	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser
	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock p2p layer into stack
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return nil }

	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser
	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock p2p layer into stack
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return nil }

	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
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
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())
	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser
	// inject mock sharder into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder
	// inject mock p2p layer into stack
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	app := TestAppConfig()
	txHandler := func(tx dto.Transaction) error { return nil }

	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
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

// peer connection handshake, happy path
func TestPeerHandshake(t *testing.T) {
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

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// define a detault tx handler call back for app
	gotCallback := false
	txHandler := func(tx dto.Transaction) error { gotCallback = true; return nil }

	// register app
	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

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

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

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
	if !mockP2PLayer.DidBroadcast {
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

// test stack controller event listener handles RECV_ShardSyncMsg correctly
func TestEventListenerHandleRECV_ShardSyncMsgEvent(t *testing.T) {
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
	msg := NewShardSyncMsg(dto.TestAnchor())
	msg.Anchor.Weight = 0xff
	// now emit RECV_ShardSyncMsg event
	events <- newControllerEvent(RECV_ShardSyncMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle shard sync
	// when peer's Anchor is heavier than local shard's Anchor

	// we should have sent the ParentRequest message from peer Anchor's parent to walk back on DAG
	// TBD
}

// stack controller listner generates RECV_NewTxBlockMsg event for unseen network message
func TestPeerListnerGeneratesEventForUnseenTxBlockMsg(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define a default tx handler call back for app
	txHandler := func(tx dto.Transaction) error { return nil }

	// register app
	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

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

// stack controller listner generates RECV_ShardSyncMsg event for unseen network message
func TestPeerListnerGeneratesEventForShardSyncMsg(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define a default tx handler call back for app
	txHandler := func(tx dto.Transaction) error { return nil }

	// register app
	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

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
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

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

// test that DLT stack does not forward a transaction that is
// rejected by application's transaction handler
func TestAppCallbackTxRejected(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define a tx handler call back for app
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
	if mockP2PLayer.DidBroadcast {
		t.Errorf("Listener frowarded an invalid network transaction")
	}
}

// DLT stack controller's runner with a registered app, happy path
// (transaction message, shutdown)
func TestStackRunner(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDbProvider())

	// inject mock p2p module into stack
	stack.p2p = p2p.TestP2PLayer("mock p2p")

	// inject mock sharding layer into stack
	sharder := NewMockSharder(stack.db)
	stack.sharder = sharder

	// inject mock endorser into stack
	endorser := NewMockEndorser(stack.db)
	stack.endorser = endorser

	// define a detault tx handler call back for app
	gotCallback := false
	txHandler := func(tx dto.Transaction) error { gotCallback = true; return nil }

	// register app
	app := TestAppConfig()
	if err := stack.Register(app.ShardId, app.Name, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

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
