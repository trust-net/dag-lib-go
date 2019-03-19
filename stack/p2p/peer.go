// Copyright 2018-2019 The trust-net Authors
// P2P Peer interface and implementation for DAG protocol library
package p2p

import (
	"errors"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/trust-net/dag-lib-go/common"
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/repo"
	"net"
//	"sync"
)

// P2P layer's wrapper for extracting Peer interface from underlying implementations
type Peer interface {
	// get identity of the peer node
	ID() []byte
	// get name of the peer node
	Name() string
	// get remove address of the peer node
	RemoteAddr() net.Addr
	// get local address in connection to the peer node
	LocalAddr() net.Addr
	// disconnect with peer node
	Disconnect()
	// connection status with peer node
	Status() int
	// a human readable representation of peer node
	String() string
	// send a message to peer node
	Send(msgId []byte, msgcode uint64, data interface{}) error
	// mark a message as seen for this peer
	Seen(msgId []byte)
	// reset seen set due to a sync
	ResetSeen()
	// read a message from peer node
	ReadMsg() (Msg, error)
	// save state during sync
	SetState(stateId int, stateData interface{}) error
	// fetch state during sync
	GetState(stateId int) interface{}
	// Shard children Q
	ShardChildrenQ() repo.Queue
	// push a transaction into stack for processing later
	ToBeFetchedStackPush(tx dto.Transaction) error
	// pop a transaction from stack for processing (nil if stack empty)
	ToBeFetchedStackPop() dto.Transaction
	// set logger
	SetLogger(logger log.Logger)
	// get logger
	Logger() log.Logger
}

const (
	// Peer connected
	Connected = 0x00
	// Peer disconnected
	Disconnected = 0x01
)

// A wrapper interface on p2p.Peer's method that we'll use in our Peer implementation,
// so that it can conveniently mocked by a test fixture for testing (basically writing testable code)
type peerDEVp2pWrapper interface {
	ID() discover.NodeID
	Name() string
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	Disconnect(reason p2p.DiscReason)
	String() string
}

// a DEVp2p based implementation of P2P layer's Peer interface
type peerDEVp2p struct {
	peer           peerDEVp2pWrapper
	rw             p2p.MsgReadWriter
	seen           *common.Set
	status         int
	states         map[int]interface{}
	shardChildrenQ repo.Queue
	txStack        []dto.Transaction
//	lock           sync.RWMutex
	logger         log.Logger
}

func NewDEVp2pPeer(peer peerDEVp2pWrapper, rw p2p.MsgReadWriter) *peerDEVp2p {
	q, err := repo.NewQueue(100)
	if err != nil {
		return nil
	}
	p := &peerDEVp2p{
		peer:           peer,
		rw:             rw,
		status:         Connected,
		seen:           common.NewSet(),
		states:         make(map[int]interface{}),
		shardChildrenQ: q,
		txStack:        []dto.Transaction{},
	}
	return p
}

func (p *peerDEVp2p) SetLogger(logger log.Logger) {
	p.logger = logger
}

func (p *peerDEVp2p) Logger() log.Logger {
	return p.logger
}

func (p *peerDEVp2p) ID() []byte {
	return p.peer.ID().Bytes()
}

func (p *peerDEVp2p) Name() string {
	return p.peer.Name()
}

func (p *peerDEVp2p) RemoteAddr() net.Addr {
	return p.peer.RemoteAddr()
}

func (p *peerDEVp2p) LocalAddr() net.Addr {
	return p.peer.LocalAddr()
}

func (p *peerDEVp2p) Disconnect() {
	p.status = Disconnected
	p.peer.Disconnect(p2p.DiscSelf)
	return
}

func (p *peerDEVp2p) Status() int {
	return p.status
}

func (p *peerDEVp2p) String() string {
	return p.peer.String()
}

func (p *peerDEVp2p) Send(msgId []byte, msgcode uint64, data interface{}) error {
	if !p.seen.Has(string(msgId)) {
		p.Seen(msgId)
		return p2p.Send(p.rw, msgcode, data)
	}
	return errors.New("seen transaction")
}

func (p *peerDEVp2p) Seen(msgId []byte) {
	if p.seen.Size() > 100 {
		for i := 0; i < 20; i += 1 {
			p.seen.Pop()
		}
	}
	p.seen.Add(string(msgId))
}

func (p *peerDEVp2p) ResetSeen() {
	p.seen = common.NewSet()
}

func (p *peerDEVp2p) ReadMsg() (Msg, error) {
	if m, err := p.rw.ReadMsg(); err != nil {
		return nil, err
	} else {
		return newMsg(&m), nil
	}
}

func (p *peerDEVp2p) SetState(stateId int, stateData interface{}) error {
	p.states[stateId] = stateData
	return nil
}

func (p *peerDEVp2p) GetState(stateId int) interface{} {
	return p.states[stateId]
}

func (p *peerDEVp2p) ShardChildrenQ() repo.Queue {
	return p.shardChildrenQ
}

func (p *peerDEVp2p) ToBeFetchedStackPush(tx dto.Transaction) error {
//	p.lock.Lock()
//	defer p.lock.Unlock()
	p.txStack = append([]dto.Transaction{tx}, p.txStack...)
	return nil
}

func (p *peerDEVp2p) ToBeFetchedStackPop() dto.Transaction {
//	p.lock.Lock()
//	defer p.lock.Unlock()
	if len(p.txStack) > 0 {
		tx := p.txStack[0]
		p.txStack = p.txStack[1:]
		return tx
	} else {
		return nil
	}
}
