package shard

import (
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"testing"
)

func TestInitiatization(t *testing.T) {
	var s Sharder
	var err error
	s, err = NewSharder(db.NewInMemDatabase())
	if s.(*sharder) == nil || err != nil {
		t.Errorf("Initiatization validation failed: %s, err: %s", s, err)
	}
}

func TestRegistration(t *testing.T) {
	s, _ := NewSharder(db.NewInMemDatabase())
	// register an app
	txHandler := func(tx *dto.Transaction) error { return nil }

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
}

func TestUnregistration(t *testing.T) {
	s, _ := NewSharder(db.NewInMemDatabase())
	// register an app
	txHandler := func(tx *dto.Transaction) error { return nil }
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

func TestHandlerUnregistered(t *testing.T) {
	s, _ := NewSharder(db.NewInMemDatabase())
	// send a mock transaction to sharder with no app registered
	if err := s.Handle(&dto.Transaction{
		ShardId: []byte("test shard"),
	}); err != nil {
		t.Errorf("Unregistered transacton handling failed: %s", err)
	}
}

func TestHandlerRegistered(t *testing.T) {
	s, _ := NewSharder(db.NewInMemDatabase())

	// register an app
	called := false
	txHandler := func(tx *dto.Transaction) error { called = true; return nil }
	s.Register([]byte("test shard"), txHandler)

	// send a mock transaction to sharder
	if err := s.Handle(&dto.Transaction{
		ShardId: []byte("test shard"),
	}); err != nil {
		t.Errorf("Registered transacton handling failed: %s", err)
	}

	// verify that callback got called
	if !called {
		t.Errorf("Sharder did not invoke transaction call back")
	}
}

func TestHandlerAppFiltering(t *testing.T) {
	s, _ := NewSharder(db.NewInMemDatabase())

	// register an app
	called := false
	txHandler := func(tx *dto.Transaction) error { called = true; return nil }
	s.Register([]byte("test shard1"), txHandler)

	// send a mock transaction to sharder from different shard
	if err := s.Handle(&dto.Transaction{
		ShardId: []byte("test shard2"),
	}); err != nil {
		t.Errorf("Unregistered transacton handling failed: %s", err)
	}

	// verify that callback did not get called
	if called {
		t.Errorf("Sharder did not filter transaction from unregistered shard")
	}
}

func TestHandlerTransactionValidation(t *testing.T) {
	s, _ := NewSharder(db.NewInMemDatabase())

	// register an app
	called := false
	txHandler := func(tx *dto.Transaction) error { called = true; return nil }
	s.Register([]byte("test shard"), txHandler)

	// send a mock transaction to sharder with missing shard ID
	if err := s.Handle(&dto.Transaction{}); err == nil {
		t.Errorf("sharder did not check for missing shard ID")
	}

	// verify that callback did not get called
	if called {
		t.Errorf("Sharder did not filter invalid transaction")
	}
}
