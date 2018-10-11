package stack

import (

)

// protocol specs
const (
	// Name should contain the official protocol name
	ProtocolName = "smithy"

	// Version should contain the version number of the protocol.
	ProtocolVersion = uint(0x01)

	// Length should contain the number of message codes used
	// by the protocol.
	ProtocolLength = uint64(2)
)

// protocol messages
const (
	// peer connection shutdown
	NodeShutdownMsgCode = uint64(0)
	// application's transaction message
	TransactionMsgCode = uint64(1)
)

// node shutdown message
type NodeShutdown struct {}

// application handshake message
type AppConfig struct {
	// public ID of the application instance (different from node ID used in p2p layer)
	AppId []byte
	// name of the application
	Name string
	// shard ID of the application (same for all nodes of application)
	ShardId []byte
	// protocol version for the shard (same for all nodes of application)
	Version uint64
}

// transaction message
type Transaction struct {
	// serialized transaction payload
	Payload []byte
	// transaction signature
	Signature []byte
	// transaction approver application instance ID
	AppId []byte
	// transaction submitter's public ID
	Submitter []byte
}
