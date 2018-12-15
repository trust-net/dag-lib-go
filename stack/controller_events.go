package stack

import ()

type eventEnum int

const (
	_                           = iota
	SEND_ShardSyncMsg eventEnum = iota
	RECV_ShardSyncMsg
	RECV_NewTxBlockMsg
	SEND_AncestorsRequestMsg
	RECV_AncestorsRequestMsg
	SEND_AncestorsResponseMsg
	RECV_ChildrenRequestMsg
	SEND_ChildrenResponseMsg
	RECV_TxBlockSyncRequestMsg
	SEND_TxBlockSyncResponseMsg
	RECV_AncestorsResponseMsg
	POP_AncestorsStack
	SEND_TxBlockSyncRequestMsg
	RECV_TxBlockSyncResponseMsg
	SEND_ChildrenRequestMsg
	RECV_ChildrenResponseMsg
	POP_FirstChild
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
