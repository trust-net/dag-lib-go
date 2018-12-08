package stack

import (
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/shard"
)

func TestAppConfig() AppConfig {
	return AppConfig{
		AppId:   []byte("test app ID"),
		ShardId: []byte("test shard"),
		Name:    "test app",
	}
}

func TestTransaction() *dto.Transaction {
	return dto.TestTransaction()
}

func TestSignedTransaction(data string) *dto.Transaction {
	tx, _ := shard.SignedShardTransaction(data)
	return tx
}

type mockEndorser struct {
	TxId            [64]byte
	Tx              *dto.Transaction
	TxHandlerCalled bool
	ReplayCalled    bool
	HandlerReturn   error
}

func (e *mockEndorser) Handle(tx *dto.Transaction) error {
	e.TxHandlerCalled = true
	e.TxId = tx.Id()
	e.Tx = tx
	return e.HandlerReturn
}

func (e *mockEndorser) Replay(txHandler func(tx *dto.Transaction) error) error {
	e.ReplayCalled = true
	if e.Tx != nil {
		return txHandler(e.Tx)
	}
	return nil
}

func NewMockEndorser() *mockEndorser {
	return &mockEndorser{
		HandlerReturn: nil,
	}
}

type mockSharder struct {
	IsRegistered    bool
	ShardId         []byte
	TxHandlerCalled bool
	TxHandler       func(tx *dto.Transaction) error
	orig            shard.Sharder
}

func (s *mockSharder) Register(shardId []byte, txHandler func(tx *dto.Transaction) error) error {
	s.IsRegistered = true
	s.ShardId = shardId
	s.TxHandler = txHandler
	return s.orig.Register(shardId, txHandler)
}

func (s *mockSharder) Unregister() error {
	s.IsRegistered = false
	s.TxHandler = nil
	return s.orig.Unregister()
}

func (s *mockSharder) Handle(tx *dto.Transaction) error {
	s.TxHandlerCalled = true
	return s.orig.Handle(tx)
}

func NewMockSharder() *mockSharder {
	db, _ := repo.NewDltDb(db.NewInMemDbProvider())
	orig, _ := shard.NewSharder(db)
	return &mockSharder{orig: orig}
}
