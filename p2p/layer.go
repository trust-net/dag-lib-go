// Copyright 2018 The trust-net Authors
// P2P Layer interface and implementation for DAG protocol library
package p2p

import (
	"github.com/ethereum/go-ethereum/p2p"
)

type P2P interface {

}

type p2pImpl struct {
	conf p2p.Config
	srv* p2p.Server
}

// create an instance of p2p layer using DEVp2p implementation
func NewDEVp2pLayer(conf p2p.Config) *p2pImpl {
	impl := &p2pImpl {
		conf: conf,
		srv: &p2p.Server{Config: conf},
	}
	return impl
}