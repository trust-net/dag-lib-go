// Copyright 2018 The trust-net Authors
// P2P Layer interface and implementation for DAG protocol library
package p2p

import (
//	"fmt"
	"math/big"
	"crypto/ecdsa"
    "crypto/sha512"
    "crypto/rand"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/go-trust-net/common"
)

type Layer interface {
	Start() error
	Self() string
	Id() []byte
	Sign(data []byte) ([]byte, error)
//	Verify(data []byte, signature []byte, peer Peer) bool
}

type Runner func(peer Peer) error

type signature struct {
	R *big.Int
	S *big.Int
}

type layerDEVp2p struct {
	conf *p2p.Config
	key *ecdsa.PrivateKey
	srv *p2p.Server
	cb Runner
	id []byte
}

func (l *layerDEVp2p) Start() error {
	return l.srv.Start()
}

func (l *layerDEVp2p) Self() string {
	return l.srv.Self().String()
}

func (l *layerDEVp2p) Id() []byte {
	return l.id
}

func (l *layerDEVp2p) Sign(data []byte) ([]byte, error) {
	s := signature{}
	var err error
	// sign the payload using SHA512 hash and ECDSA signature
	hash := sha512.Sum512(data)
	if s.R,s.S, err = ecdsa.Sign(rand.Reader, l.key, hash[:]); err != nil {
		return nil, err
	}
	if signature, err := common.Serialize(s); err != nil {
		return nil, err
	} else {
		return signature, nil
	}
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
func NewDEVp2pLayer(c Config, cb Runner) (*layerDEVp2p, error) {
	conf, err := c.toDEVp2pConfig()
	if err != nil {
		return nil, err
	}
	impl := &layerDEVp2p {
		conf: conf,
		cb: cb,
		key: conf.PrivateKey,
	}
	impl.conf.Protocols = impl.makeDEVp2pProtocols(c)
	impl.srv = &p2p.Server{Config: *impl.conf}
	impl.id = impl.srv.Self().ID.Bytes()
	return impl, nil
}