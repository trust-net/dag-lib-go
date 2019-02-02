// Copyright 2018-2019 The trust-net Authors
// P2P message interface and implementation for DAG protocol library
package p2p

import (
	"github.com/ethereum/go-ethereum/p2p"
)

// wrapper interface for underlying p2p message implementation
type Msg interface {
	Code() uint64
	Decode(val interface{}) error
	String() string
	Discard() error
}

func newMsg(p2pMsg *p2p.Msg) *msg {
	return &msg{
		p2pMsg: p2pMsg,
	}
}

type msg struct {
	p2pMsg *p2p.Msg
}

func (m *msg) Code() uint64 {
	return m.p2pMsg.Code
}

func (m *msg) Decode(val interface{}) error {
	return m.p2pMsg.Decode(val)
}

func (m *msg) String() string {
	return m.p2pMsg.String()
}

func (m *msg) Discard() error {
	return m.p2pMsg.Discard()
}
