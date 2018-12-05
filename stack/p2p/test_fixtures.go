// Copyright 2018 The trust-net Authors
// Mock DEVp2p implementations for use as test fixtures
package p2p

import (
	"net"
	"errors"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
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

type mockMsgReadWriter struct {
	ReadCount  int
	WriteCount int
	msgs []p2p.Msg
}

func TestConn() *mockMsgReadWriter {
	return &mockMsgReadWriter{}
}

func (m *mockMsgReadWriter) NextMsg(msgcode uint64, data interface{}) {
	size, r, _ := rlp.EncodeToReader(data)
	msg := p2p.Msg {Code: msgcode, Size: uint32(size), Payload: r}
	m.msgs = append(m.msgs, msg)
}

func (m *mockMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	m.ReadCount += 1
	if len(m.msgs) > 0 {
		msg := m.msgs[0]
		m.msgs = m.msgs[1:]
		return msg, nil
	}
	return p2p.Msg{}, errors.New("no more messages")
}

func (m *mockMsgReadWriter) WriteMsg(p2p.Msg) error {
	m.WriteCount += 1
	return nil
}

func TestP2PLayer(name string) *mockP2P {
	return &mockP2P {
		Name: name,
		ID: []byte("some random ID"),
	}
}

type mockP2P struct {
	IsStarted bool
	IsStopped bool
	DidBroadcast bool
	Name string
	ID []byte
}

func (p2p *mockP2P) Start() error {
	p2p.IsStarted = true
	return nil
}

func (p2p *mockP2P) Disconnect(peer Peer) {
	return
}

func (p2p *mockP2P) Stop() {
	p2p.IsStopped = true
	return
}

func (p2p *mockP2P) Self() string {
	return p2p.Name
}

func (p2p *mockP2P) Id() []byte {
	return p2p.ID
}

func (p2p *mockP2P) Sign(data []byte) ([]byte, error) {
	return []byte("signature"), nil
}

func (p2p *mockP2P) Verify(payload, sign, id []byte) bool {
	return true
}

func (p2p *mockP2P) Broadcast(msgId []byte, msgcode uint64, data interface{}) error {
	p2p.DidBroadcast = true
	return nil
}

// implements peerDEVp2pWrapper interface, so can be used interchangeabily with DEVp2p.Peer 
type mockPeer struct {
	IdCount int
	NameCount int
	RemoteCount int
	LocalCount int
	DisconnectCount int
	StringCount int
}

func (p* mockPeer) ID() discover.NodeID {
	p.IdCount += 1
	return discover.NodeID{}
}
func (p* mockPeer) Name() string {
	p.NameCount +=1
	return ""
}
func (p* mockPeer) RemoteAddr() net.Addr {
	p.RemoteCount += 1
	return nil
}
func (p* mockPeer) LocalAddr() net.Addr {
	p.LocalCount += 1
	return nil
}
func (p* mockPeer) Disconnect(reason p2p.DiscReason) {
	p.DisconnectCount += 1
}
func (p* mockPeer) String() string {
	p.StringCount += 1
	return ""
}

func TestDEVp2pPeer(name string) *p2p.Peer {
	nodeId, _ := discover.BytesID([]byte(name))
	return p2p.NewPeer(nodeId, name, nil)
}

func TestMockPeer(name string) *mockPeer {
	return &mockPeer{}
}