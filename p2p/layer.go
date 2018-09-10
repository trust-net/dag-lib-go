// Copyright 2018 The trust-net Authors
// P2P Layer interface and implementation for DAG protocol library
package p2p

import (
//	"fmt"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type Layer interface {

}

type Runner func(peer Peer) error

type Config struct {
	// TODO: change this to simple json param, and then create actual ECDSA key at run time
	// This field must be set to a valid secp256k1 private key.
//	PrivateKey *ecdsa.PrivateKey `toml:"-"`
	PrivateKey string

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int

	// Name sets the node name of this server.
	Name string `toml:"-"`

	// TODO: change this to simple json string, and then create actual discover.Node at run time
	// BootstrapNodes are used to establish connectivity
	// with the rest of the network.
//	BootstrapNodes []*discover.Node
	BootstrapNodes []string

	// Name should contain the official protocol name,
	// often a three-letter word.
	ProtocolName string

	// Version should contain the version number of the protocol.
	ProtocolVersion uint

	// Length should contain the number of message codes used
	// by the protocol.
	ProtocolLength uint64
//
//	// Runner is called in a new groutine when the protocol has been
//	// negotiated with a peer. It should read and write messages from
//	// rw. The Payload for each message must be fully consumed.
//	//
//	// The peer connection is closed when Start returns. It should return
//	// any protocol-level error (such as an I/O error) that is
//	// encountered.
//	Runner Runner

	// If ListenAddr is set to a non-nil address, the server
	// will listen for incoming connections.
	//
	// If the port is zero, the operating system will pick a port. The
	// ListenAddr field will be updated with the actual address when
	// the server is started.
	ListenAddr string
	Port string

	// TODO: change this to simple json boolean (or string), and then instantiate nat.Interface at run time
	// If set to a non-nil value, the given NAT port mapper
	// is used to make the listening port available to the
	// Internet.
//	NAT nat.Interface `toml:",omitempty"`
	NAT bool
}

func (c *Config) key() *ecdsa.PrivateKey {
	// TODO
	return nil
}

func (c *Config) nat() nat.Interface {
	// TODO
	return nil
}

func (c *Config) bootnodes() []*discover.Node {
	// TODO
	return nil
}

func (c *Config) toDEVp2pConfig() p2p.Config {
	conf := p2p.Config {
		MaxPeers:       c.MaxPeers,
		PrivateKey:     c.key(),
		Name:           c.Name,
		ListenAddr:     c.ListenAddr + ":" + c.Port,
		NAT: 	        c.nat(),
//		Protocols:      c.protocols(),
		BootstrapNodes: c.bootnodes(),
	}
	return conf
}

type layerDEVp2p struct {
	conf p2p.Config
	srv* p2p.Server
	cb Runner
}

// we are just wrapping the callback to hide the DEVp2p specific details
func (l *layerDEVp2p) runner(dPeer *p2p.Peer, dRw p2p.MsgReadWriter) error {
	peer := NewDEVp2pPeer(dPeer, dRw)
	return l.cb(peer)
}

func (l *layerDEVp2p) makeDEVp2pProtocols(conf Config) []p2p.Protocol {
	proto := p2p.Protocol {
		Name: conf.ProtocolName,
		Version: conf.ProtocolVersion,
		Length: conf.ProtocolLength,
		Run: l.runner,
	}
	return []p2p.Protocol{proto}
}

// create an instance of p2p layer using DEVp2p implementation
func NewDEVp2pLayer(conf Config, cb Runner) *layerDEVp2p {
	impl := &layerDEVp2p {
		conf: conf.toDEVp2pConfig(),
		cb: cb,
	}
	impl.conf.Protocols = impl.makeDEVp2pProtocols(conf)
	impl.srv = &p2p.Server{Config: impl.conf}
	return impl
}