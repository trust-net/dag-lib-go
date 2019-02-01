// Copyright 2018-2019 The trust-net Authors
// P2P Layer interface and implementation for DAG protocol library
package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"math/big"
	"sync"
)

type Layer interface {
	// sign a transaction Anchor
	Anchor(a *dto.Anchor) error
	Start() error
	Stop()
	Disconnect(peer Peer)
	Self() string
	Id() []byte
	Sign(data []byte) ([]byte, error)
	Verify(data, sign, id []byte) bool
	Broadcast(msgId []byte, msgcode uint64, data interface{}) error
}

type Runner func(peer Peer) error

type signature struct {
	R *big.Int
	S *big.Int
}

type layerDEVp2p struct {
	conf  *p2p.Config
	key   *ecdsa.PrivateKey
	srv   *p2p.Server
	cb    Runner
	id    []byte
	peers map[string]Peer
	lock  sync.RWMutex
}

func (l *layerDEVp2p) Anchor(a *dto.Anchor) error {
	if a == nil {
		return errors.New("cannot sign nil anchor")
	}
	// force update anchor's node ID with this node
	a.NodeId = l.Id()
	if signature, err := l.Sign(a.Bytes()); err != nil {
		return err
	} else {
		a.Signature = signature
		return nil
	}
}

func (l *layerDEVp2p) Start() error {
	return l.srv.Start()
}

func (l *layerDEVp2p) Disconnect(peer Peer) {
	// remove the peer from peer map
	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.peers, string(peer.ID()))
	peer.Disconnect()
}

func (l *layerDEVp2p) Stop() {
	// disconnect from all connected peers
	for _, peer := range l.peers {
		peer.Disconnect()
	}
	l.srv.Stop()
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
	// sign the payload using SHA256 hash and ECDSA signature
	hash := sha256.Sum256(data)
	if s.R, s.S, err = ecdsa.Sign(rand.Reader, l.key, hash[:]); err != nil {
		return nil, err
	}
	return append(s.R.Bytes(), s.S.Bytes()...), nil
}

func (l *layerDEVp2p) Verify(payload, sign, id []byte) bool {
	// extract submitter's key
	key := crypto.ToECDSAPub(id)
	if key == nil || key.X == nil {
		return false
	}

	// regenerate signature parameters
	s := signature{
		R: &big.Int{},
		S: &big.Int{},
	}
	if len(sign) == 65 {
		sign = sign[1:]
	}
	if len(sign) != 64 {
		return false
	}
	s.R.SetBytes(sign[0:32])
	s.S.SetBytes(sign[32:64])

	// we want to validate the hash of the payload
	hash := sha256.Sum256(payload)
	// validate signature of payload
	return ecdsa.Verify(key, hash[:], s.R, s.S)
}

func (l *layerDEVp2p) Broadcast(msgId []byte, msgcode uint64, data interface{}) error {
	// walk through list of peers and send messages
	for _, peer := range l.peers {
		if err := peer.Send(msgId, msgcode, data); err != nil {
			// skip
//			return err
		}
	}
	return nil
}

// we are just wrapping the callback to hide the DEVp2p specific details
func (l *layerDEVp2p) runner(dPeer *p2p.Peer, dRw p2p.MsgReadWriter) error {
	peer := NewDEVp2pPeer(dPeer, dRw)
	// add the peer to layer's peers map
	l.lock.Lock()
	l.peers[string(peer.ID())] = peer
	l.lock.Unlock()
	defer func() {
		l.lock.Lock()
		delete(l.peers, string(peer.ID()))
		l.lock.Unlock()
	}()
	return l.cb(peer)
}

func (l *layerDEVp2p) makeDEVp2pProtocols(conf Config) []p2p.Protocol {
	proto := p2p.Protocol{
		Name:    conf.ProtocolName,
		Version: conf.ProtocolVersion,
		Length:  conf.ProtocolLength,
		Run:     l.runner,
	}
	return []p2p.Protocol{proto}
}

// create an instance of p2p layer using DEVp2p implementation
func NewDEVp2pLayer(c Config, cb Runner) (*layerDEVp2p, error) {
	conf, err := c.toDEVp2pConfig()
	if err != nil {
		return nil, err
	}
	impl := &layerDEVp2p{
		conf:  conf,
		cb:    cb,
		key:   conf.PrivateKey,
		id:    crypto.FromECDSAPub(&conf.PrivateKey.PublicKey),
		peers: make(map[string]Peer),
	}
	impl.conf.Protocols = impl.makeDEVp2pProtocols(c)
	impl.srv = &p2p.Server{Config: *impl.conf}
	return impl, nil
}
