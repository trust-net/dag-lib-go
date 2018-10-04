// Copyright 2018 The trust-net Authors
// P2P Layer interface and implementation for DAG protocol library
package p2p

import (
//	"fmt"
	"github.com/ethereum/go-ethereum/p2p"
)

type Layer interface {
	Start() error
	Self() string
}

type Runner func(peer Peer) error


type layerDEVp2p struct {
	conf *p2p.Config
	srv *p2p.Server
	cb Runner
}

func (l *layerDEVp2p) Start() error {
	return l.srv.Start()
}


func (l *layerDEVp2p) Self() string {
	return l.srv.Self().String()
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
	}
	impl.conf.Protocols = impl.makeDEVp2pProtocols(c)
	impl.srv = &p2p.Server{Config: *impl.conf}
	return impl, nil
}