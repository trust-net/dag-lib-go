package stack

import ()

type eventEnum int

const (
	_                           = iota
	SEND_ShardSyncMsg eventEnum = iota
	RECV_ShardSyncMsg
	RECV_NewTxBlockMsg
	SEND_ShardAncestorRequestMsg
	RECV_ShardAncestorRequestMsg
	SEND_ShardAncestorResponseMsg
	RECV_ShardAncestorResponseMsg
	RECV_ShardChildrenRequestMsg
	SEND_ShardChildrenResponseMsg
	RECV_ShardChildrenResponseMsg
	RECV_TxShardChildRequestMsg
	RECV_TxShardChildResponseMsg
	RECV_ForceShardSyncMsg
	RECV_SubmitterWalkUpRequestMsg
	RECV_SubmitterWalkUpResponseMsg
	POP_ShardChild
	SHUTDOWN
)

type controllerEvent struct {
	code eventEnum
	data interface{}
}

func newControllerEvent(code eventEnum, data interface{}) controllerEvent {
	return controllerEvent{
		code: code,
		data: data,
	}
}
