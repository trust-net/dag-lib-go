// Copyright 2018 The trust-net Authors
// Mock DEVp2p implementations for use as test fixtures
package p2p

import (
	"net"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type mockMsgReadWriter struct {
	readCount  int
	writeCount int
}

func testConn() *mockMsgReadWriter {
	return &mockMsgReadWriter{}
}

func (m *mockMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	m.readCount += 1
	return p2p.Msg{}, nil
}

func (m *mockMsgReadWriter) WriteMsg(p2p.Msg) error {
	m.writeCount += 1
	return nil
}

// implements peerDEVp2pWrapper interface, so can be used interchangeabily with DEVp2p.Peer 
type mockPeer struct {
	idCount int
	nameCount int
	remoteCount int
	localCount int
	disconnectCount int
	stringCount int
}

func (p* mockPeer) ID() discover.NodeID {
	p.idCount += 1
	return discover.NodeID{}
}
func (p* mockPeer) Name() string {
	p.nameCount +=1
	return ""
}
func (p* mockPeer) RemoteAddr() net.Addr {
	p.remoteCount += 1
	return nil
}
func (p* mockPeer) LocalAddr() net.Addr {
	p.localCount += 1
	return nil
}
func (p* mockPeer) Disconnect(reason p2p.DiscReason) {
	p.disconnectCount += 1
}
func (p* mockPeer) String() string {
	p.stringCount += 1
	return ""
}

func testDEVp2pPeer(name string) *p2p.Peer {
	return p2p.NewPeer(discover.NodeID{}, name, nil)
}

func testMockPeer(name string) *mockPeer {
	return &mockPeer{}
}