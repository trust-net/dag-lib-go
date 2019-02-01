// Copyright 2018-2019 The trust-net Authors
// Configuration for P2P Layer initialization for DAG protocol library
package p2p

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"math/big"
	"os"
)

type ECDSAKey struct {
	Curve string
	X, Y  []byte
	D     []byte
}

type Config struct {
	// path to private key for p2p layer node
	KeyFile string `json:"key_file"       gencodec:"required"`

	// type of private key for p2p layer node ("ECDSA_S256")
	KeyType string `json:"key_type"       gencodec:"required"`

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int `json:"max_peers"       gencodec:"required"`

	// Name sets the node name of this server.
	Name string `json:"node_name"       gencodec:"required"`

	// Bootnodes are used to establish connectivity
	// with the rest of the network.
	Bootnodes []string `json:"boot_nodes"`

	// Name should contain the official protocol name,
	// often a three-letter word.
	ProtocolName string `json:"proto_name"       gencodec:"required"`

	// Version should contain the version number of the protocol.
	ProtocolVersion uint `json:"proto_ver"       gencodec:"required"`

	// Length should contain the number of message codes used
	// by the protocol.
	ProtocolLength uint64

	// If ListenAddr is set to a non-nil address, the server
	// will listen for incoming connections.
	ListenAddr string `json:"listen_addr"`

	// If the port is zero, the operating system will pick a port.
	Port string `json:"listen_port"`

	// If set to true, the listening port is made available to the
	// Internet.
	NAT bool
}

func (c *Config) key() (*ecdsa.PrivateKey, error) {
	// basic validation checks
	if len(c.KeyFile) == 0 {
		return nil, errors.New("missing 'key_file' parameter")
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
					return nil, err
				} else {
					nodekey := new(ecdsa.PrivateKey)
					nodekey.PublicKey.Curve = crypto.S256()
					nodekey.D = new(big.Int)
					nodekey.D.SetBytes(ecdsaKey.D)
					nodekey.PublicKey.X = new(big.Int)
					nodekey.PublicKey.X.SetBytes(ecdsaKey.X)
					nodekey.PublicKey.Y = new(big.Int)
					nodekey.PublicKey.Y.SetBytes(ecdsaKey.Y)
					return nodekey, nil
				}
			} else {
				return nil, errors.New("failed to read KeyFile")
			}
		} else {
			// generate new secret key and persist to file
			nodekey, _ := crypto.GenerateKey()
			ecdsaKey := ECDSAKey{
				Curve: "S256",
				X:     nodekey.X.Bytes(),
				Y:     nodekey.Y.Bytes(),
				D:     nodekey.D.Bytes(),
			}
			if data, err := json.Marshal(ecdsaKey); err == nil {
				if file, err := os.Create(c.KeyFile); err == nil {
					file.Write(data)
				} else {
					return nil, errors.New("failed to save KeyFile")
				}
			} else {
				return nil, err
			}
			return nodekey, nil
		}
	default:
		return nil, errors.New("missing or unsupported 'key_type' parameter")
	}
}

func (c *Config) nat() nat.Interface {
	if c.NAT {
		return nat.Any()
	} else {
		return nil
	}
}

func (c *Config) listenAddr() string {
	if len(c.Port) != 0 {
		return c.ListenAddr + ":" + c.Port
	} else {
		return c.ListenAddr
	}
}

func (c *Config) bootnodes() []*discover.Node {
	// parse bootnodes from config, if present
	if c.Bootnodes != nil {
		bootnodes := make([]*discover.Node, 0, len(c.Bootnodes))
		for _, bootnode := range c.Bootnodes {
			if enode, err := discover.ParseNode(bootnode); err == nil {
				bootnodes = append(bootnodes, enode)
			}
		}
		if len(bootnodes) > 0 {
			return bootnodes
		}
	}
	// we either did not have any bootnode config, or none of entry was valid
	return nil
}

func (c *Config) toDEVp2pConfig() (*p2p.Config, error) {
	key, err := c.key()
	switch {
	case key == nil:
		return nil, err
	case c.MaxPeers < 1:
		return nil, errors.New("'max_peers' must be non zero")
	case len(c.ProtocolName) == 0:
		return nil, errors.New("missing 'proto_name' parameter")
	case len(c.Name) == 0:
		return nil, errors.New("missing 'node_name' parameter")
	}
	conf := p2p.Config{
		MaxPeers:       c.MaxPeers,
		PrivateKey:     key,
		Name:           c.Name,
		ListenAddr:     c.listenAddr(),
		NAT:            c.nat(),
		BootstrapNodes: c.bootnodes(),
	}
	return &conf, nil
}
