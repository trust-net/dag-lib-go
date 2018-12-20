package stack

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
)

// protocol specs
const (
	// Name should contain the official protocol name
	ProtocolName = "smithy"

	// Version should contain the version number of the protocol.
	ProtocolVersion = uint(0x01)
)

// protocol messages
const (
	// peer connection shutdown
	NodeShutdownMsgCode uint64 = iota
	// application's transaction message
	TransactionMsgCode
	// shard level sync message
	ShardSyncMsgCode
	// ancestors request message to walk back for shard's DAG
	ShardAncestorRequestMsgCode
	// ancestors response message
	ShardAncestorResponseMsgCode
	// childrens request for a known hash to populate shard's DAG
	ShardChildrenRequestMsgCode
	// childrens response for a known hash
	ShardChildrenResponseMsgCode
	// ProtocolLength should contain the number of message codes used
	// by the protocol.
	ProtocolLength
)

// node shutdown message
type NodeShutdown struct{}

// application configuration
type AppConfig struct {
	// public ID of the application instance (same as node ID used in p2p layer)
	AppId []byte
	// name of the application
	Name string
	// shard ID of the application (same for all nodes of application)
	ShardId []byte
}

type ShardAncestorRequestMsg struct {
	StartHash    [64]byte
	MaxAncestors uint64
}

func (m *ShardAncestorRequestMsg) Id() []byte {
	id := []byte("ShardAncestorRequestMsg")
	return append(id, m.StartHash[:]...)
}

func (m *ShardAncestorRequestMsg) Code() uint64 {
	return ShardAncestorRequestMsgCode
}

type ShardChildrenRequestMsg struct {
	Parent [64]byte
}

func (m *ShardChildrenRequestMsg) Id() []byte {
	id := []byte("ShardChildrenRequestMsg")
	return append(id, m.Parent[:]...)
}

func (m *ShardChildrenRequestMsg) Code() uint64 {
	return ShardChildrenRequestMsgCode
}

type ShardAncestorResponseMsg struct {
	StartHash [64]byte
	Ancestors [][64]byte
}

func (m *ShardAncestorResponseMsg) Id() []byte {
	id := []byte("ShardAncestorResponseMsg")
	return append(id, m.StartHash[:]...)
}

func (m *ShardAncestorResponseMsg) Code() uint64 {
	return ShardAncestorResponseMsgCode
}

type ShardChildrenResponseMsg struct {
	Parent   [64]byte
	Children [][64]byte
}

func (m *ShardChildrenResponseMsg) Id() []byte {
	id := []byte("ShardChildrenResponseMsg")
	return append(id, m.Parent[:]...)
}

func (m *ShardChildrenResponseMsg) Code() uint64 {
	return ShardChildrenResponseMsgCode
}

type ShardSyncMsg struct {
	Anchor *dto.Anchor
}

func (m *ShardSyncMsg) Id() []byte {
	return append([]byte("ShardSyncMsg"), m.Anchor.ShardId...)
}

func (m *ShardSyncMsg) Code() uint64 {
	return ShardSyncMsgCode
}

func NewShardSyncMsg(anchor *dto.Anchor) *ShardSyncMsg {
	return &ShardSyncMsg{
		Anchor: anchor,
	}
}
