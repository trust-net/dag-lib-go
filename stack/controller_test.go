package stack

import (
    "testing"
    "errors"
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
	peerHandler := func (id []byte) bool {return true}
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
	peerHandler := func (id []byte) bool {return true}
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
	peerHandler := func (id []byte) bool {return true}
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
	peerHandler := func (id []byte) bool {return true}
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
	peerHandler := func (id []byte) bool {return true}
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
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	app := TestAppConfig()
	peerHandler := func (id []byte) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	if err := stack.Submit(TestTransaction()); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}
	if !p2p.DidBroadcast {
		t.Errorf("Transaction did not get broadcast to peers")
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
	if !p2p.IsStarted {
		t.Errorf("Controller did not start p2p layer")
	}
}

// stop of controller, happy path
func TestStop(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
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

// Application's handler callback test, happy path (app ID is accepted by application)
func TestAppCallbackPeerAccepted(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define peer handler call back for app
	peerHandlerCbCount := 0
	peerHandler := func (id []byte) bool {
		peerHandlerCbCount += 1
		// we trust all and accept all :)
		return true
	}

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

	// setup mock connection to send a signed transaction followed by clean shutdown
	tx := TestSignedTransaction("test payload")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// now call stack's listener
	if err := stack.listener(peer); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// app's peer handler should have been called
	if peerHandlerCbCount != 1 {
		t.Errorf("app peer validation handler not called")
	}

	// app's transaction handler should have been called
	if !txHandlerCb {
		t.Errorf("app peer transaction handler not called")
	}

	// we should have attempted to read messaged 2 times
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should have broadcasted message
	if !mockP2PLayer.DidBroadcast {
		t.Errorf("Listener did not froward valid network transaction")
	}
}

// Application's handler callback test, when app ID is NOT accepted by application
func TestAppCallbackPeerNotAccepted(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define peer handler call back for app
	peerHandler := func (id []byte) bool {
		// we trust no one and accept none :)
		return false
	}

	// define a tx handler call back for app
	txHandler := func (tx *Transaction) error {
		return nil
	}

	// register app
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// setup mock connection to send a signed transaction followed by clean shutdown
	tx := TestSignedTransaction("test payload")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// now call stack's listener
	if err := stack.listener(peer); err != nil {
		t.Errorf("Listener quit on peer app rejected by app: %s", err)
	}

	// we should have attempted to read messaged 2 times
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should not have broadcasted message
	if mockP2PLayer.DidBroadcast {
		t.Errorf("Listener frowarded network transaction from unaccepted peer")
	}
}

// test that DLT stack does not forward a transaction that is
// rejected by application's transaction handler
func TestAppCallbackTxRejected(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define peer handler call back for app
	peerHandler := func (id []byte) bool {
		// we everyone
		return true
	}

	// define a tx handler call back for app
	txHandler := func (tx *Transaction) error {
		// we reject all transactions
		return errors.New("trust no one")
	}

	// register app
	if err := stack.Register(TestAppConfig(), peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// setup mock connection to send a signed transaction followed by clean shutdown
	tx := TestSignedTransaction("test payload")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// now call stack's listener
	if err := stack.listener(peer); err != nil {
		t.Errorf("Listener quit on transaction rejected by app: %s", err)
	}

	// we should have attempted to read 2 messages
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should not have broadcasted message
	if mockP2PLayer.DidBroadcast {
		t.Errorf("Listener frowarded an invalid network transaction")
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
	peerHandler := func (id []byte) bool { return true }

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
	//    transaction message
	//    node shutdown message
	mockConn.NextMsg(TransactionMsgCode, &Transaction{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// now simulate a new connection session
	if err := stack.runner(peer); err != nil {
		t.Errorf("Peer connection has error: %s", err)
	}

	// all messages should have been consumed from peer
	if mockConn.ReadCount != 2 {
		t.Errorf("Did not expect %d messages consumed from peer", mockConn.ReadCount)
	}

	// no messages should have been sent to peer (yet)
	if mockConn.WriteCount != 0 {
		t.Errorf("Did not expect %d messages sent to peer", mockConn.WriteCount)
	}
}
