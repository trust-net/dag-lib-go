// Copyright 2018 The trust-net Authors
// Mock DEVp2p implementations for use as test fixtures
package p2p

import (
	"net"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func TestConfig() Config {
	return Config{
		KeyFile: "key_file.json",
		KeyType: "ECDSA_S256",
		MaxPeers: 1,
		ProtocolName: "test-protocol",
		Name: "test node",
	}
}

type MockMsgReadWriter struct {
	ReadCount  int
	WriteCount int
}

func TestConn() *MockMsgReadWriter {
	return &MockMsgReadWriter{}
}

func (m *MockMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	m.ReadCount += 1
	return p2p.Msg{}, nil
}

func (m *MockMsgReadWriter) WriteMsg(p2p.Msg) error {
	m.WriteCount += 1
	return nil
}

func TestP2PLayer(name string) *MockP2P {
	return &MockP2P {
		Name: name,
	}
}

type MockP2P struct {
	IsStarted bool
	Name string
}

func (p2p *MockP2P) Start() error {
	p2p.IsStarted = true
	return nil
}

func (p2p *MockP2P) Self () string {
	return p2p.Name
}

// implements peerDEVp2pWrapper interface, so can be used interchangeabily with DEVp2p.Peer 
type MockPeer struct {
	IdCount int
	NameCount int
	RemoteCount int
	LocalCount int
	DisconnectCount int
	StringCount int
}

func (p* MockPeer) ID() discover.NodeID {
	p.IdCount += 1
	return discover.NodeID{}
}
func (p* MockPeer) Name() string {
	p.NameCount +=1
	return ""
}
func (p* MockPeer) RemoteAddr() net.Addr {
	p.RemoteCount += 1
	return nil
}
func (p* MockPeer) LocalAddr() net.Addr {
	p.LocalCount += 1
	return nil
}
func (p* MockPeer) Disconnect(reason p2p.DiscReason) {
	p.DisconnectCount += 1
}
func (p* MockPeer) String() string {
	p.StringCount += 1
	return ""
}

func TestDEVp2pPeer(name string) *p2p.Peer {
	return p2p.NewPeer(discover.NodeID{}, name, nil)
}

func TestMockPeer(name string) *MockPeer {
	return &MockPeer{}
}