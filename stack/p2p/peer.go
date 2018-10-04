// Copyright 2018 The trust-net Authors
// P2P Peer interface and implementation for DAG protocol library
package p2p

import (
	"net"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
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
	Send(msgcode uint64, data interface{}) error
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
	peer peerDEVp2pWrapper
	rw p2p.MsgReadWriter
	status int
}

func NewDEVp2pPeer(peer peerDEVp2pWrapper, rw p2p.MsgReadWriter) *peerDEVp2p {
	return &peerDEVp2p{
		peer: peer,
		rw: rw,
		status: Connected,
	}
}

func (p* peerDEVp2p) ID() []byte {
	return p.peer.ID().Bytes()
}

func (p* peerDEVp2p) Name() string {
	return p.peer.Name()
}

func (p* peerDEVp2p) RemoteAddr() net.Addr {
	return p.peer.RemoteAddr()
}

func (p* peerDEVp2p) LocalAddr() net.Addr {
	return p.peer.LocalAddr()
}

func (p* peerDEVp2p) Disconnect()  {
	p.status = Disconnected
	p.peer.Disconnect(p2p.DiscSelf)
	return
}

func (p* peerDEVp2p) Status() int  {
	return p.status
}

func (p* peerDEVp2p) String() string {
	return p.peer.String()
}

func (p* peerDEVp2p) Send(msgcode uint64, data interface{}) error {
	return p2p.Send(p.rw, msgcode, data)
}
