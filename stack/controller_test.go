package stack

import (
    "testing"
	"github.com/trust-net/dag-lib-go/db"
)

func TestInitiatization(t *testing.T) {
	var stack DLT
	var err error
	testDb := db.NewInMemDatabase()
	stack, err = NewDltStack(testDb)
	if stack.(*dlt) == nil || err != nil {
		t.Errorf("Initiatization validation failed, c: %s, err: %s", stack, err)
	}
	if stack.(*dlt).db != testDb {
		t.Errorf("Stack does not have correct DB reference expected: %s, actual: %s", testDb, stack.(*dlt).db)
	}
}

func TestRegister(t *testing.T) {
	stack, _ := NewDltStack(db.NewInMemDatabase())
	app := AppConfig{}
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	if string(stack.app.NodeId) != string(app.NodeId) || string(stack.app.ShardId) != string(app.ShardId) || stack.app.NodeName != app.NodeName {
		t.Errorf("App configuration not initialized correctly")
	}
	if stack.peerHandler == nil || stack.txHandler == nil {
		t.Errorf("Callback methods not initialized correctly")
	}
}

func TestPreRegistered(t *testing.T) {
	stack, _ := NewDltStack(db.NewInMemDatabase())
	app := AppConfig{}
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
	}
	
	if err := stack.Register(AppConfig{}, peerHandler, txHandler); err == nil {
		t.Errorf("Registration did not check for already registered")
	}
}

func TestUnRegister(t *testing.T) {
	stack, _ := NewDltStack(db.NewInMemDatabase())
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(AppConfig{}, peerHandler, txHandler); err != nil {
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
	stack, _ := NewDltStack(db.NewInMemDatabase())
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for unregistered")
	}
}

func testTransaction() *Transaction {
	return &Transaction {
		Payload: []byte("test data"),
		Signature: []byte("test signature"),
		NodeId: []byte("test node ID"),
		Submitter: []byte("test submitter"),
	}
}

func TestSubmitNilValues(t *testing.T) {
	stack, _ := NewDltStack(db.NewInMemDatabase())
	app := AppConfig{}
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	if err := stack.Submit(nil); err == nil {
		t.Errorf("Transaction submission did not check for nil transaction")
	}
	tx := testTransaction()
	tx.Payload = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil payload")
	}
	tx = testTransaction()
	tx.Signature = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil signature")
	}
	tx = testTransaction()
	tx.NodeId = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil node ID")
	}
	tx = testTransaction()
	tx.Submitter = nil
	if err := stack.Submit(tx); err == nil {
		t.Errorf("Transaction submission did not check for nil submitter ID")
	}
}

func TestSubmit(t *testing.T) {
	stack, _ := NewDltStack(db.NewInMemDatabase())
	app := AppConfig{}
	peerHandler := func (app AppConfig) bool {return true}
	txHandler := func (tx *Transaction) error {return nil}
	
	if err := stack.Register(app, peerHandler, txHandler); err != nil {
		t.Errorf("Registration failed, err: %s", err)
		return
	}
	if err := stack.Submit(testTransaction()); err != nil {
		t.Errorf("Transaction submission failed, err: %s", err)
	}
}
