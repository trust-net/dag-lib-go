// Copyright 2018 The trust-net Authors
// Configuration for P2P Layer initialization for DAG protocol library
package p2p

import (
	"math/big"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type ECDSAKey struct{
	Curve string
	X, Y []byte
	D []byte
}

type Config struct {
	// TODO: change this to simple json param, and then create actual ECDSA key at run time
	// This field must be set to a valid secp256k1 private key.
//	PrivateKey *ecdsa.PrivateKey `toml:"-"`
	PrivateKey ECDSAKey

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
	// basic validation checks
	if c.PrivateKey.Curve != "S256" || len(c.PrivateKey.D) == 0 || len(c.PrivateKey.X) == 0 || len(c.PrivateKey.Y) == 0 {
		return nil
	}
	key := new(ecdsa.PrivateKey)
	key.PublicKey.Curve = crypto.S256()
	key.D = new(big.Int)
	key.D.SetBytes(c.PrivateKey.D) 
	key.PublicKey.X = new(big.Int)
	key.PublicKey.X.SetBytes(c.PrivateKey.X)
	key.PublicKey.Y = new(big.Int)
	key.PublicKey.Y.SetBytes(c.PrivateKey.Y)
	return key
}

func (c *Config) nat() nat.Interface {
	// TODO
	return nil
}

func (c *Config) bootnodes() []*discover.Node {
	// TODO
	return nil
}

func (c *Config) toDEVp2pConfig() *p2p.Config {
	conf := p2p.Config {
		MaxPeers:       c.MaxPeers,
		PrivateKey:     c.key(),
		Name:           c.Name,
		ListenAddr:     c.ListenAddr + ":" + c.Port,
		NAT: 	        c.nat(),
//		Protocols:      c.protocols(),
		BootstrapNodes: c.bootnodes(),
	}
	return &conf
}
