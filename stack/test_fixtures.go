package stack

import (
	devp2p "github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/endorsement"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"net"
)

func TestAppConfig() AppConfig {
	return AppConfig{
		AppId:   []byte("test app ID"),
		ShardId: []byte("test shard"),
		Name:    "test app",
	}
}

func TestTransaction() dto.Transaction {
	return dto.TestTransaction()
}

func TestAnchoredTransaction(a *dto.Anchor, data string) dto.Transaction {
	tx, _ := shard.SignedShardTransaction(data)
	tx.Self().TxAnchor = a
	return tx
}

func TestSignedTransaction(data string) dto.Transaction {
	tx, _ := shard.SignedShardTransaction(data)
	return tx
}

type mockEndorser struct {
	TxId            [64]byte
	Tx              dto.Transaction
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

func (e *mockEndorser) Approve(tx dto.Transaction) error {
	e.ApproverCalled = true
	e.TxId = tx.Id()
	return e.orig.Approve(tx)
}

func (e *mockEndorser) Handle(tx dto.Transaction) error {
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
	IsRegistered     bool
	ShardId          []byte
	AnchorCalled     bool
	SyncAnchorCalled bool
	AncestorsCalled  bool
	ApproverCalled   bool
	TxHandlerCalled  bool
	TxHandler        func(tx dto.Transaction) error
	orig             shard.Sharder
}

func (s *mockSharder) Register(shardId []byte, txHandler func(tx dto.Transaction) error) error {
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

func (s *mockSharder) SyncAnchor(shardId []byte) *dto.Anchor {
	s.SyncAnchorCalled = true
	return s.orig.SyncAnchor(shardId)
}

func (s *mockSharder) Ancestors(startHash [64]byte, max uint64) [][64]byte {
	s.AncestorsCalled = true
	return s.orig.Ancestors(startHash, max)
}

func (s *mockSharder) Approve(tx dto.Transaction) error {
	s.ApproverCalled = true
	return s.orig.Approve(tx)
}

func (s *mockSharder) Handle(tx dto.Transaction) error {
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

type mockPeer struct {
	peer             p2p.Peer
	IDCalled         bool
	NameCalled       bool
	RemoteAddrCalled bool
	LocalAddrCalled  bool
	DisconnectCalled bool
	SendCalled       bool
	SendMsgId        []byte
	SendMsgCode      uint64
	SendMsg          interface{}
	SeenCalled       bool
	ReadMsgCalled    bool
}

func NewMockPeer(mockConn devp2p.MsgReadWriter) *mockPeer {
	mockP2pPeer := p2p.TestMockPeer("test peer")
	return &mockPeer{
		peer: p2p.NewDEVp2pPeer(mockP2pPeer, mockConn),
	}
}

func (p *mockPeer) ID() []byte {
	p.IDCalled = true
	return p.peer.ID()
}

func (p *mockPeer) Name() string {
	p.NameCalled = true
	return p.peer.Name()
}

func (p *mockPeer) RemoteAddr() net.Addr {
	p.RemoteAddrCalled = true
	return p.peer.RemoteAddr()
}

func (p *mockPeer) LocalAddr() net.Addr {
	p.LocalAddrCalled = true
	return p.peer.LocalAddr()
}

func (p *mockPeer) Disconnect() {
	p.DisconnectCalled = true
	p.peer.Disconnect()
	return
}

func (p *mockPeer) Status() int {
	return p.peer.Status()
}

func (p *mockPeer) String() string {
	return p.peer.String()
}

func (p *mockPeer) Send(msgId []byte, msgcode uint64, data interface{}) error {
	p.SendCalled = true
	p.SendMsgId = msgId
	p.SendMsgCode = msgcode
	p.SendMsg = data
	return p.peer.Send(msgId, msgcode, data)
}

func (p *mockPeer) Seen(msgId []byte) {
	p.SeenCalled = true
	p.peer.Seen(msgId)
}

func (p *mockPeer) ReadMsg() (p2p.Msg, error) {
	p.ReadMsgCalled = true
	return p.peer.ReadMsg()
}
