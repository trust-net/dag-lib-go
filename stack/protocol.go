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
	ShardId      []byte
	StartHash    [64]byte
	MaxAncestors int
}

func (m *ShardAncestorRequestMsg) Id() []byte {
	return append(m.ShardId, m.StartHash[:]...)
}

func (m *ShardAncestorRequestMsg) Code() uint64 {
	return ShardAncestorRequestMsgCode
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
