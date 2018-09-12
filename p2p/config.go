// Copyright 2018 The trust-net Authors
// Configuration for P2P Layer initialization for DAG protocol library
package p2p

import (
	"os"
	"math/big"
	"crypto/ecdsa"
	"encoding/json"
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
	// path to private key for p2p layer node
	KeyFile string

	// type of private key for p2p layer node ("ECDSA_S256")
	KeyType string

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
	if len(c.KeyFile) == 0 {
		return nil
	}
	switch c.KeyType {
		case "ECDSA_S256":
			// read the keyfile, if present, else create a new key and persist
			if file, err := os.Open(c.KeyFile); err == nil {
				// source the secret key from file
				data := make([]byte, 1024)
				if count, err := file.Read(data); err == nil && count <= 1024 {
					data = data[:count]
					ecdsaKey := ECDSAKey{}
					if err := json.Unmarshal(data, &ecdsaKey); err != nil {
						return nil
					} else {
						nodekey := new(ecdsa.PrivateKey)
						nodekey.PublicKey.Curve = crypto.S256()
						nodekey.D = new(big.Int)
						nodekey.D.SetBytes(ecdsaKey.D) 
						nodekey.PublicKey.X = new(big.Int)
						nodekey.PublicKey.X.SetBytes(ecdsaKey.X)
						nodekey.PublicKey.Y = new(big.Int)
						nodekey.PublicKey.Y.SetBytes(ecdsaKey.Y)
						return nodekey
					}
				} else {
					return nil
				}
			} else {
				// generate new secret key and persist to file
				nodekey, _ := crypto.GenerateKey()
				ecdsaKey := ECDSAKey {
					Curve: "S256",
					X: nodekey.X.Bytes(),
					Y: nodekey.Y.Bytes(),
					D: nodekey.D.Bytes(),
				}
				if data, err := json.Marshal(ecdsaKey); err == nil {
					if file, err := os.Create(c.KeyFile); err == nil {
						file.Write(data)
					} else {
						return nil
					}
				} else {
					return nil
				}
				return nodekey
			}
		default:
			return nil
	}
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
