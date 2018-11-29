package stack

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
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
	return dto.TestSignedTransaction(data)
}

type mockEndorser struct {
	TxId            []byte
	Tx              *dto.Transaction
	TxHandlerCalled bool
	ReplayCalled    bool
	HandlerReturn   error
}

func (e *mockEndorser) Handle(tx *dto.Transaction) error {
	e.TxHandlerCalled = true
	e.TxId = tx.Signature
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
}

func (s *mockSharder) Register(shardId []byte, txHandler func(tx *dto.Transaction) error) error {
	s.IsRegistered = true
	s.ShardId = shardId
	s.TxHandler = txHandler
	return nil
}

func (s *mockSharder) Unregister() error {
	s.IsRegistered = false
	s.TxHandler = nil
	return nil
}

func (s *mockSharder) Handle(tx *dto.Transaction) error {
	s.TxHandlerCalled = true
	if s.TxHandler != nil {
		return s.TxHandler(tx)
	} else {
		return nil
	}
}

func NewMockSharder() *mockSharder {
	return &mockSharder{}
}
