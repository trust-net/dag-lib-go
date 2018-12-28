// Copyright 2018 The trust-net Authors
// P2P Layer interface and implementation for DAG protocol library
package p2p

import (
	//	"fmt"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/go-trust-net/common"
	"math/big"
	"sync"
)

type Layer interface {
	// populate a transaction Anchor
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

func uint64ToBytes(value uint64) []byte {
	var byte8 [8]byte
	binary.BigEndian.PutUint64(byte8[:], value)
	return byte8[:]
}

func (l *layerDEVp2p) Anchor(a *dto.Anchor) error {
	if a == nil {
		return errors.New("cannot sign nil anchor")
	}
	// update anchor's node ID with this node
	a.NodeId = l.Id()
	// sign the anchor and fill in Anchor signature
	payload := []byte{}
	payload = append(payload, a.ShardId...)
	payload = append(payload, a.NodeId...)
	payload = append(payload, a.Submitter...)
	payload = append(payload, a.ShardParent[:]...)
	for _, uncle := range a.ShardUncles {
		payload = append(payload, uncle[:]...)
	}
	payload = append(payload, uint64ToBytes(a.ShardSeq)...)
	payload = append(payload, uint64ToBytes(a.Weight)...)

	// sign the test payload using SHA512 hash and ECDSA private key
	s := signature{}
	hash := sha512.Sum512(payload)
	s.R, s.S, _ = ecdsa.Sign(rand.Reader, l.key, hash[:])
	if sign, err := common.Serialize(s); err != nil {
		return err
	} else {
		a.Signature = sign
	}
	return nil
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
	// sign the payload using SHA512 hash and ECDSA signature
	hash := sha512.Sum512(data)
	if s.R, s.S, err = ecdsa.Sign(rand.Reader, l.key, hash[:]); err != nil {
		return nil, err
	}
	if signature, err := common.Serialize(s); err != nil {
		return nil, err
	} else {
		return signature, nil
	}
}

func (l *layerDEVp2p) Verify(payload, sign, id []byte) bool {
	// extract submitter's key
	key := crypto.ToECDSAPub(id)
	if key == nil {
		return false
	}

	// regenerate signature parameters
	s := signature{}
	if err := common.Deserialize(sign, &s); err != nil {
		// Failed to parse signature
		return false
	}
	// we want to validate the hash of the payload
	hash := sha512.Sum512(payload)
	// validate signature of payload
	return ecdsa.Verify(key, hash[:], s.R, s.S)
}

func (l *layerDEVp2p) Broadcast(msgId []byte, msgcode uint64, data interface{}) error {
	// walk through list of peers and send messages
	for _, peer := range l.peers {
		if err := peer.Send(msgId, msgcode, data); err != nil {
			return err
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
