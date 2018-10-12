package stack

import (
    "testing"
    "errors"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/p2p"
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
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	// our app's ID should be same as p2p node's ID
	if string(stack.app.AppId) != string(stack.p2p.Id()) || string(stack.app.ShardId) != string(app.ShardId) || stack.app.Name != app.Name || stack.app.Version != app.Version {
		t.Errorf("App configuration not initialized correctly")
	}
	if stack.txHandler == nil {
		t.Errorf("Callback methods not initialized correctly")
	}
}

// attempt to register app when already registered
func TestPreRegistered(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := AppConfig{}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	
	if err := stack.Register(TestAppConfig(), txHandler); err == nil {
		t.Errorf("Registration did not check for already registered")
	}
}

// unregister a previously registered application
func TestUnRegister(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(TestAppConfig(), txHandler); err != nil {
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
	txHandler := func (tx *Transaction) error {return nil}	
	if err := stack.Register(app, txHandler); err != nil {
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
	txHandler := func (tx *Transaction) error {return nil}
	if err := stack.Register(app, txHandler); err != nil {
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
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, txHandler); err != nil {
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

	// now call stack's listener
	if err := stack.listener(peer); err != nil {
		t.Errorf("Listener failed to process transaction as headless: %s", err)
	}

	// we should have read 2 messages from peer
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should have broadcasted message
	if !mockP2PLayer.DidBroadcast {
		t.Errorf("Listener did not froward network transaction as headless")
	}

	// we should have marked the message as seen for stack
	if !stack.isSeen(tx.Signature) {
		t.Errorf("Listener did not mark the transaction as seen while headless")
	}
}

// statck controller listener with a previously seen message
func TestPeerListenerSeenMessage(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	mockP2PLayer := p2p.TestP2PLayer("mock p2p")
	stack.p2p = mockP2PLayer

	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// define a default tx handler call back for app
	txHandler := func (tx *Transaction) error { return nil }

	// register app
	if err := stack.Register(TestAppConfig(), txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}

	// setup mock connection to send a signed transaction followed by clean shutdown
	tx := TestSignedTransaction("test payload")
	mockConn.NextMsg(TransactionMsgCode, tx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// mark the message seen with stack
	stack.isSeen(tx.Signature)

	// now call stack's listener
	if err := stack.listener(peer); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// we should have attempted to read messaged 2 times
	if mockConn.ReadCount != 2 {
		t.Errorf("Listener read %d messages", mockConn.ReadCount)
	}

	// we should not have broadcasted seen message
	if mockP2PLayer.DidBroadcast {
		t.Errorf("Listener frowarded a seen transaction")
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

	// define a tx handler call back for app
	txHandler := func (tx *Transaction) error {
		// we reject all transactions
		return errors.New("trust no one")
	}

	// register app
	if err := stack.Register(TestAppConfig(), txHandler); err != nil {
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

	// we should have marked the message as seen for stack
	if !stack.isSeen(tx.Signature) {
		t.Errorf("Listener did not mark the transaction as seen while headless")
	}
}

// DLT stack controller's runner with a registered app, happy path 
// (app handshake, transaction message, shutdown)
func TestStackRunner(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	stack.p2p = p2p.TestP2PLayer("mock p2p")

	// define a detault tx handler call back for app
	txHandler := func (tx *Transaction) error { return nil}

	// register app
	if err := stack.Register(TestAppConfig(), txHandler); err != nil {
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
