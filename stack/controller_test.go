package stack

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/p2p"
//	devP2P "github.com/ethereum/go-ethereum/p2p"
)

// initialize DLT stack and validate
func TestInitiatization(t *testing.T) {
	var stack DLT
	var err error
	testDb := db.NewInMemDatabase()
	stack, err = NewDltStack(p2p.TestConfig(), testDb)
	if stack.(*dlt) == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", stack, err)
	}
	if stack.(*dlt).db != testDb {
		t.Errorf("Stack does not have correct DB reference expected: %s, actual: %s", testDb, stack.(*dlt).db)
	}
	if len(stack.(*dlt).p2p.Self()) == 0 {
		t.Errorf("Stack does not have correct p2p layer")
	}
}

// register application
func TestRegister(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := TestAppConfig()
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	// our app's ID should be same as p2p node's ID
	if string(stack.app.AppId) != string(stack.p2p.Id()) || string(stack.app.ShardId) != string(app.ShardId) || stack.app.Name != app.Name || stack.app.Version != app.Version {
		t.Errorf("App configuration not initialized correctly")
	}
	if stack.peerHandler == nil || stack.txHandler == nil {
		t.Errorf("Callback methods not initialized correctly")
	}
}

// attempt to register app when already registered
func TestPreRegistered(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := AppConfig{}
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err == nil {
		t.Errorf("Registration did not check for already registered")
	}
}

// unregister a previously registered application
func TestUnRegister(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	if err := stack.Unregister(); err != nil {
		t.Errorf("Unregistration failed, err: %s", err)
	}
	if stack.app != nil {
		t.Errorf("App configuration not cleared during unregister")
	}
	if stack.peerHandler != nil || stack.txHandler != nil {
		t.Errorf("Callback methods not cleared during unregister")
	}
}

// try submitting a transaction without application being registered first
func TestSubmitUnregistered(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for unregistered")
	}
}

// try submitting a transaction with nil/missing values
func TestSubmitNilValues(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := TestAppConfig()
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}

	// try submitting a nil transaction
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for nil transaction")
	}

	// try submitting nil payload
	tx := TestTransaction()
	tx.Payload = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil payload")
	}

	// signature should automatically be created when submitted
	tx = TestTransaction()
	tx.Signature = nil
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission did not create signature")
	}

	// app ID should automatically be updated from node's ID
	tx = TestTransaction()
	tx.AppId = nil
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission did not update app ID")
	}

	// submitter ID needs to be non-null
	tx = TestTransaction()
	tx.Submitter = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil submitter ID")
	}
}

// try submitting a transaction with fake app ID, it should get corrected
func TestSubmitAppIdNoMatch(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := TestAppConfig()
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	tx := TestTransaction()
	tx.AppId = []byte("some random app ID")
	if err := stack.Submit(tx); err != nil {
		t.Errorf("Transaction submission did ignore fake app ID")
	}
	if string(tx.AppId) != string(stack.p2p.Id()) {
		t.Errorf("Transaction submission did not update correct app ID")
	}
}

// transaction submission, happy path
func TestSubmit(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := TestAppConfig()
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	if err := stack.Submit(TestTransaction()); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}
}

// start of controller, happy path
func TestStart(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	if err := stack.Start(); err != nil || !p2p.IsStarted {
		t.Errorf("Controller failed to start: %s", err)
	}
}

// peer connection handshake, happy path
func TestPeerHandshake(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)
	
	// invoke handshake
	if err := stack.handshake(peer); err != nil {
		t.Errorf("Handshake failed, err: %s", err)
	}
	
	// for iteration #1 no handshake message should be exchanged
	if mockConn.WriteCount != 0 {
		t.Errorf("Handshake exchanged %d messages", mockConn.WriteCount)
	}
}

// statck controller listener with no app registered, happy path
func TestPeerListenerNoApp(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)
	
	// setup mock connection to send a transaction message
	mockConn.NextMsg(TransactionMsgCode, &Transaction{})
	
	// invoke listener
	if err := stack.listener(peer); err == nil {
		t.Errorf("Listener did not check for app registration")
	}

	// we should have read a message from peer
	if mockConn.ReadCount != 1 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}
}

// Application's peer handler callback test, happy path (app ID matches node ID)
func TestAppPeerHandlerCallback(t *testing.T) {
// actually following is not true, App ID would be of the application instance's node ID, but
// peer's ID would be of the node that is forwarding this message to us 


	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	stack.p2p = p2p.TestP2PLayer("mock p2p")

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define peer handler call back for app
	peerHandlerCbCount := 0
	peerHandler := func (app AppConfig) bool {
		// we want to validate that app config matches p2p node's ID
		peerHandlerCbCount += 1
		return string(app.AppId) == string(peer.ID())
	}

	// define a detault tx handler call back for app
	txHandler := func (tx *Transaction) error { return nil}

	// register app
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// setup mock connection to send a application handshake
	peerAppConfig := TestAppConfig()
	// make sure app's ID matches p2p node's ID
	peerAppConfig.AppId = mockP2pPeer.ID().Bytes()
	mockConn.NextMsg(AppConfigMsgCode, &peerAppConfig)

	// now simulate a new peer app connection
	// (we don't call listener directly, since handshake may happen anywhere upon new connection)
	stack.runner(peer)

	// app's peer handler should have been called
	if peerHandlerCbCount != 1 {
		t.Errorf("app peer validation handler not called")
	}

	// we should have attempted to read messaged 2 times
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}
}

// Application's peer handler callback test when app ID does not match node ID
func TestAppPeerHandlerNodeIdMismatch(t *testing.T) {
// actually following is not true, App ID would be of the application instance's node ID, but
// peer's ID would be of the node that is forwarding this message to us 

//	// create an instance of stack controller
//	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
//
//	// inject mock p2p module into stack
//	stack.p2p = p2p.TestP2PLayer("mock p2p")
//
//	// build a mock peer
//	mockP2pPeer := p2p.TestMockPeer("test peer")
//	mockConn := p2p.TestConn()
//	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)
//
//	// define peer handler call back for app
//	appIdMatchedNodeId := false
//	peerHandler := func (app AppConfig) bool {
//		appIdMatchedNodeId = string(app.AppId) == string(peer.ID())
//		// we want to validate that app config matches p2p node's ID
//		return appIdMatchedNodeId
//	}
//
//	// define a detault tx handler call back for app
//	txHandler := func (tx *Transaction) error { return nil}
//
//	// register app
//	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
//		t.Errorf("Registration failed, err: %s", err)
//	}
//
//	// setup mock connection to send a application handshake
//	peerAppConfig := TestAppConfig()
//	// make sure app's ID does not matches p2p node's ID
//	peerAppConfig.AppId = []byte("some randome node ID")
//	mockConn.NextMsg(AppConfigMsgCode, &peerAppConfig)
//
//	// now simulate a new peer app connection
//	// (we don't call listener directly, since handshake may happen anywhere upon new connection)
//	stack.runner(peer)
//
//	// app's ID should have matched node ID (even though we sent fake)
//	if !appIdMatchedNodeId {
//		t.Errorf("app ID did not match node ID")
//	}
}

// Application's transaction handler callback test, happy path
func TestAppTxHandlerCallback(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	stack.p2p = p2p.TestP2PLayer("mock p2p")

	// define an default peer handler call back for app
	peerHandler := func (app AppConfig) bool { return true }

	// define a tx handler call back for app
	txHandlerCb := false
	txHandler := func (tx *Transaction) error {
		txHandlerCb = true
		return nil
	}

	// register app
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// setup mock connection to send a transaction message
	mockConn.NextMsg(TransactionMsgCode, &Transaction{})

	// now invoke stack's listener
	stack.listener(peer)


	// app's transaction handler should have been called
	if !txHandlerCb {
		t.Errorf("app peer transaction handler not called")
	}
}

// DLT stack controller's runner with a registered app, happy path 
// (app handshake, transaction message, shutdown)
func TestStackRunner(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	stack.p2p = p2p.TestP2PLayer("mock p2p")

	// define an default peer handler call back for app
	peerHandler := func (app AppConfig) bool { return true }

	// define a detault tx handler call back for app
	txHandler := func (tx *Transaction) error { return nil}

	// register app
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// setup mock connection to send following messages:
	//    application handshake
	//    transaction message
	//    node shutdown message
	peerAppConfig := TestAppConfig()
	mockConn.NextMsg(AppConfigMsgCode, &peerAppConfig)
	mockConn.NextMsg(TransactionMsgCode, &Transaction{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// now simulate a new connection session
	if err := stack.runner(peer); err != nil {
		t.Errorf("Peer connection has error: %s", err)
	}

	// all messages should have been consumed from peer
	if mockConn.ReadCount != 3 {
		t.Errorf("Did not expect %d messages consumed from peer", mockConn.ReadCount)
	}

	// no messages should have been sent to peer (yet)
	if mockConn.WriteCount != 0 {
		t.Errorf("Did not expect %d messages sent to peer", mockConn.WriteCount)
	}
}
