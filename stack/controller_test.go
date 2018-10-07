package stack

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/p2p"
//	devP2P "github.com/ethereum/go-ethereum/p2p"
)

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

func TestRegister(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := TestAppConfig()
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	if string(stack.app.AppId) != string(app.AppId) || string(stack.app.ShardId) != string(app.ShardId) || stack.app.Name != app.Name || stack.app.Version != app.Version {
		t.Errorf("App configuration not initialized correctly")
	}
	if stack.peerHandler == nil || stack.txHandler == nil {
		t.Errorf("Callback methods not initialized correctly")
	}
}

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

func TestSubmitUnregistered(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for unregistered")
	}
}

func TestSubmitNilValues(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	app := TestAppConfig()
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for nil transaction")
	}
	tx := TestTransaction()
	tx.Payload = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil payload")
	}
	tx = TestTransaction()
	tx.Signature = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil signature")
	}
	tx = TestTransaction()
	tx.AppId = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil app ID")
	}
	tx = TestTransaction()
	tx.Submitter = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil submitter ID")
	}
}

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
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for app ID match")
	}
}

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


func TestStart(t *testing.T) {
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())
	p2p := p2p.TestP2PLayer("mock p2p")
	stack.p2p = p2p
	if err := stack.Start(); err != nil || !p2p.IsStarted {
		t.Errorf("Controller failed to start: %s", err)
	}
}

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

func TestAppPeerHandlerCallback(t *testing.T) {
	// create an instance of stack controller
	stack, _ := NewDltStack(p2p.TestConfig(), db.NewInMemDatabase())

	// inject mock p2p module into stack
	stack.p2p = p2p.TestP2PLayer("mock p2p")

	// define peer handler call back for app
	peerHandlerCb := false
	peerHandler := func (app AppConfig) bool {
		peerHandlerCb = true
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
	// build a mock peer
	mockP2pPeer := p2p.TestMockPeer("test peer")
	mockConn := p2p.TestConn()
	peer := p2p.NewDEVp2pPeer(mockP2pPeer, mockConn)

	// setup mock connection to send a application handshake followed by a transaction message
	peerAppConfig := TestAppConfig()
	mockConn.NextMsg(AppConfigMsgCode, &peerAppConfig)
	mockConn.NextMsg(TransactionMsgCode, &Transaction{})

	// now simulate a new peer app connection
	stack.runner(peer)

	// app's peer handler should have been called
	if !peerHandlerCb {
		t.Errorf("app peer validation handler not called")
	}

	// app's transaction handler should have been called
	if !txHandlerCb {
		t.Errorf("app peer transaction handler not called")
	}
}
