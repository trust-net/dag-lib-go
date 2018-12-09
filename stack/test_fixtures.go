package stack

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/endorsement"
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

func TestAnchoredTransaction(a *dto.Anchor, data string) *dto.Transaction {
	tx, _ := shard.SignedShardTransaction(data)
	tx.ShardId = a.ShardId
	tx.ShardParent = a.ShardParent
	tx.ShardSeq = a.ShardSeq
	return tx
}

func TestSignedTransaction(data string) *dto.Transaction {
	tx, _ := shard.SignedShardTransaction(data)
	return tx
}

type mockEndorser struct {
	TxId            [64]byte
	Tx              *dto.Transaction
	TxHandlerCalled bool
	AnchorCalled    bool
	ApproverCalled  bool
	HandlerReturn   error
	orig            endorsement.Endorser
}

func (e *mockEndorser) Anchor(a *dto.Anchor) error {
	e.AnchorCalled = true
	return e.orig.Anchor(a)
}

func (e *mockEndorser) Approve(tx *dto.Transaction) error {
	e.ApproverCalled = true
	e.TxId = tx.Id()
	return e.orig.Approve(tx)
}

func (e *mockEndorser) Handle(tx *dto.Transaction) error {
	e.TxHandlerCalled = true
	e.TxId = tx.Id()
	e.Tx = tx
	if e.HandlerReturn != nil {
		return e.HandlerReturn
	} else {
		return e.orig.Handle(tx)
	}
}

func (e *mockEndorser) Reset() {
	*e = mockEndorser{orig: e.orig}
}

func NewMockEndorser(db repo.DltDb) *mockEndorser {
	orig, _ := endorsement.NewEndorser(db)
	return &mockEndorser{
		HandlerReturn: nil,
		orig:          orig,
	}
}

type mockSharder struct {
	IsRegistered    bool
	ShardId         []byte
	AnchorCalled    bool
	ApproverCalled  bool
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

func (s *mockSharder) Anchor(a *dto.Anchor) error {
	s.AnchorCalled = true
	return s.orig.Anchor(a)
}

func (s *mockSharder) Approve(tx *dto.Transaction) error {
	s.ApproverCalled = true
	return s.orig.Approve(tx)
}

func (s *mockSharder) Handle(tx *dto.Transaction) error {
	s.TxHandlerCalled = true
	return s.orig.Handle(tx)
}

func (s *mockSharder) Reset() {
	*s = mockSharder{orig: s.orig}
}

func NewMockSharder(db repo.DltDb) *mockSharder {
	//	db, _ := repo.NewDltDb(db.NewInMemDbProvider())
	orig, _ := shard.NewSharder(db)
	return &mockSharder{orig: orig}
}
