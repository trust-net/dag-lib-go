// Copyright 2018 The trust-net Authors
// P2P Peer interface and implementation for DAG protocol library
package p2p

import (
	"github.com/ethereum/go-ethereum/p2p"
)


type Peer interface {
	
}

type peerDEVp2p struct {
	peer *p2p.Peer
	rw *p2p.MsgReadWriter
}

func NewDEVp2pPeer(peer *p2p.Peer, rw p2p.MsgReadWriter) *peerDEVp2p {
	return &peerDEVp2p{
		peer: peer,
		rw: &rw,
	}
}