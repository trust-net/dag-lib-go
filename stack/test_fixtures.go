package stack

import (
	devp2p "github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/endorsement"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"github.com/trust-net/dag-lib-go/stack/shard"
	"github.com/trust-net/dag-lib-go/stack/state"
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
	TxId                 [64]byte
	Tx                   dto.Transaction
	TxHandlerCalled      bool
	TxUpdateCalled       bool
	KnownShardsTxsCalled bool
	ReplaceCalled        bool
	ValidateCalled         bool
	ApproverCalled       bool
	HandlerReturn        error
	orig                 endorsement.Endorser
}

func (e *mockEndorser) Validate(r *dto.TxRequest) error {
	e.ValidateCalled = true
	return e.orig.Validate(r)
}

func (e *mockEndorser) Approve(tx dto.Transaction) error {
	e.ApproverCalled = true
	e.TxId = tx.Id()
	return e.orig.Approve(tx)
}

func (e *mockEndorser) Handle(tx dto.Transaction) (int, error) {
	e.TxHandlerCalled = true
	e.TxId = tx.Id()
	e.Tx = tx
	if e.HandlerReturn != nil {
		return endorsement.ERR_INVALID, e.HandlerReturn
	} else {
		return e.orig.Handle(tx)
	}
}

func (e *mockEndorser) Update(tx dto.Transaction) error {
	e.TxUpdateCalled = true
	return e.orig.Update(tx)
}

func (e *mockEndorser) KnownShardsTxs(submitter []byte, seq uint64) (shards [][]byte, txs [][64]byte) {
	e.KnownShardsTxsCalled = true
	return e.orig.KnownShardsTxs(submitter, seq)
}

func (e *mockEndorser) Replace(tx dto.Transaction) error {
	e.ReplaceCalled = true
	return e.orig.Replace(tx)
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
	LockStateCalled   bool
	UnlockStateCalled bool
	CommitStateCalled bool
	IsRegistered      bool
	ShardId           []byte
	AnchorCalled      bool
	SyncAnchorCalled  bool
	AncestorsCalled   bool
	ChildrenCalled    bool
	ApproverCalled    bool
	TxHandlerCalled   bool
	GetStateCalled    bool
	GetStateKey       []byte
	FlushCalled       bool
	TxHandler         func(tx dto.Transaction, state state.State) error
	orig              shard.Sharder
}

func (s *mockSharder) LockState() error {
	s.LockStateCalled = true
	// lock world state
	return s.orig.LockState()
}

func (s *mockSharder) UnlockState() {
	s.UnlockStateCalled = true
	// unlock world state
	s.orig.UnlockState()
}

func (s *mockSharder) CommitState(tx dto.Transaction) error {
	s.CommitStateCalled = true
	// transaction processed successfully, persist world state
	return s.orig.CommitState(tx)
}

func (s *mockSharder) Register(shardId []byte, txHandler func(tx dto.Transaction, state state.State) error) error {
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

func (s *mockSharder) Children(parent [64]byte) [][64]byte {
	s.ChildrenCalled = true
	return s.orig.Children(parent)
}

func (s *mockSharder) Approve(tx dto.Transaction) error {
	s.ApproverCalled = true
	return s.orig.Approve(tx)
}

func (s *mockSharder) Handle(tx dto.Transaction) error {
	s.TxHandlerCalled = true
	return s.orig.Handle(tx)
}

func (s *mockSharder) GetState(key []byte) (*state.Resource, error) {
	s.GetStateCalled = true
	s.GetStateKey = key
	return s.orig.GetState(key)
}

func (s *mockSharder) Flush(shardId []byte) error {
	s.FlushCalled = true
	return s.orig.Flush(shardId)
}

func (s *mockSharder) Reset() {
	*s = mockSharder{orig: s.orig}
}

func NewMockSharder(dltDb repo.DltDb) *mockSharder {
	//	db, _ := repo.NewDltDb(db.NewInMemDbProvider())
	orig, _ := shard.NewSharder(dltDb, db.NewInMemDbProvider())
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
	ResetSeenCalled  bool
	//	states           map[int]interface{}
	GetStateCalled            bool
	SetStateCalled            bool
	ShardChildrenQCallCount   int
	ToBeFetchedStackPushCount int
	ToBeFetchedStackPopCount  int
}

func NewMockPeer(mockConn devp2p.MsgReadWriter) *mockPeer {
	mockP2pPeer := p2p.TestMockPeer("test peer")
	m := &mockPeer{
		peer: p2p.NewDEVp2pPeer(mockP2pPeer, mockConn),
		//		states: make(map[int]interface{}),
	}
	m.peer.SetLogger(log.NewLogger("test peer"))
	return m
}

func (p *mockPeer) SetLogger(logger log.Logger) {
	p.peer.SetLogger(logger)
}

func (p *mockPeer) Logger() log.Logger {
	return p.peer.Logger()
}

func (p *mockPeer) Reset() {
	*p = mockPeer{
		peer: p.peer,
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

func (p *mockPeer) ResetSeen() {
	p.ResetSeenCalled = true
	p.peer.ResetSeen()
}

func (p *mockPeer) ReadMsg() (p2p.Msg, error) {
	p.ReadMsgCalled = true
	return p.peer.ReadMsg()
}

func (p *mockPeer) SetState(stateId int, stateData interface{}) error {
	p.SetStateCalled = true
	//	p.states[stateId] = stateData
	//	return nil
	return p.peer.SetState(stateId, stateData)
}

func (p *mockPeer) GetState(stateId int) interface{} {
	p.GetStateCalled = true
	//	return p.states[stateId]
	return p.peer.GetState(stateId)
}

func (p *mockPeer) ShardChildrenQ() repo.Queue {
	p.ShardChildrenQCallCount += 1
	return p.peer.ShardChildrenQ()
}

func (p *mockPeer) ToBeFetchedStackPush(tx dto.Transaction) error {
	p.ToBeFetchedStackPushCount += 1
	return p.peer.ToBeFetchedStackPush(tx)
}

func (p *mockPeer) ToBeFetchedStackPop() dto.Transaction {
	p.ToBeFetchedStackPopCount += 1
	return p.peer.ToBeFetchedStackPop()
}
