// Copyright 2018-2019 The trust-net Authors
// Trust-Net Protocol Messages
package stack

import (
	"github.com/trust-net/dag-lib-go/common"
	"github.com/trust-net/dag-lib-go/stack/dto"
)

// protocol specs
const (
	// Name should contain the official protocol name
	ProtocolName = "smithy/iter_8"

	// Version should contain the version number of the protocol
	// 2MSB Major . 2LSB Minor
	ProtocolVersion = uint(0x00000008)
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
	// transaction and its shard DAG descendents request
	TxShardChildRequestMsgCode
	// transaction and its shard DAG descendents response
	TxShardChildResponseMsgCode
	// force a shard sync during app registration
	ForceShardSyncMsgCode
	// submitter history walk up request message
	SubmitterWalkUpRequestMsgCode
	// submitter history walk up response message
	SubmitterWalkUpResponseMsgCode
	// submitter history walk down request message
	SubmitterProcessDownRequestMsgCode
	// submitter history walk down response message
	SubmitterProcessDownResponseMsgCode
	// notify remote node to flush shard due to double spend
	ForceShardFlushMsgCode
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
	ShardId []byte
	Anchor  *dto.Anchor
}

func (m *ShardSyncMsg) Id() []byte {
	id := []byte("ShardSyncMsg")
	id = append(id, m.ShardId...)
	return append(id, m.Anchor.Signature...)
}

func (m *ShardSyncMsg) Code() uint64 {
	return ShardSyncMsgCode
}

func NewShardSyncMsg(shardId []byte, anchor *dto.Anchor) *ShardSyncMsg {
	return &ShardSyncMsg{
		ShardId: shardId,
		Anchor:  anchor,
	}
}

type SubmitterWalkUpRequestMsg struct {
	Submitter []byte
	Seq       uint64
}

func (m *SubmitterWalkUpRequestMsg) Id() []byte {
	id := []byte("SubmitterWalkUpRequestMsg")
	id = append(id, common.Uint64ToBytes(m.Seq)...)
	return append(id, m.Submitter...)
}

func (m *SubmitterWalkUpRequestMsg) Code() uint64 {
	return SubmitterWalkUpRequestMsgCode
}

func NewSubmitterWalkUpRequestMsg(req *dto.TxRequest) *SubmitterWalkUpRequestMsg {
	return &SubmitterWalkUpRequestMsg{
		Submitter: req.SubmitterId,
		Seq:       req.SubmitterSeq - 1,
	}
}

type SubmitterWalkUpResponseMsg struct {
	Submitter    []byte
	Seq          uint64
	Transactions [][64]byte
	Shards       [][]byte
}

func (m *SubmitterWalkUpResponseMsg) Id() []byte {
	id := []byte("SubmitterWalkUpResponseMsg")
	id = append(id, common.Uint64ToBytes(m.Seq)...)
	return append(id, m.Submitter...)
}

func (m *SubmitterWalkUpResponseMsg) Code() uint64 {
	return SubmitterWalkUpResponseMsgCode
}

func NewSubmitterWalkUpResponseMsg(req *SubmitterWalkUpRequestMsg) *SubmitterWalkUpResponseMsg {
	return &SubmitterWalkUpResponseMsg{
		Submitter:    req.Submitter,
		Seq:          req.Seq,
		Transactions: nil,
		Shards:       nil,
	}
}

type SubmitterProcessDownRequestMsg struct {
	Submitter []byte
	Seq       uint64
}

func (m *SubmitterProcessDownRequestMsg) Id() []byte {
	id := []byte("SubmitterProcessDownRequestMsg")
	id = append(id, common.Uint64ToBytes(m.Seq)...)
	return append(id, m.Submitter...)
}

func (m *SubmitterProcessDownRequestMsg) Code() uint64 {
	return SubmitterProcessDownRequestMsgCode
}

func NewSubmitterProcessDownRequestMsg(resp *SubmitterWalkUpResponseMsg) *SubmitterProcessDownRequestMsg {
	return &SubmitterProcessDownRequestMsg{
		Submitter: resp.Submitter,
		Seq:       resp.Seq + 1,
	}
}

type SubmitterProcessDownResponseMsg struct {
	Submitter []byte
	Seq       uint64
	TxBytes   [][]byte
}

func (m *SubmitterProcessDownResponseMsg) Id() []byte {
	id := []byte("SubmitterProcessDownResponseMsg")
	id = append(id, common.Uint64ToBytes(m.Seq)...)
	return append(id, m.Submitter...)
}

func (m *SubmitterProcessDownResponseMsg) Code() uint64 {
	return SubmitterProcessDownResponseMsgCode
}

func NewSubmitterProcessDownResponseMsg(req *SubmitterProcessDownRequestMsg, txs []dto.Transaction) *SubmitterProcessDownResponseMsg {
	txBytes := [][]byte{}
	for _, tx := range txs {
		if bytes, err := tx.Serialize(); err != nil {
			return nil
		} else {
			txBytes = append(txBytes, bytes)
		}
	}
	return &SubmitterProcessDownResponseMsg{
		Submitter: req.Submitter,
		Seq:       req.Seq,
		TxBytes:   txBytes,
	}
}

type ForceShardSyncMsg struct {
	ShardId []byte
	Anchor  *dto.Anchor
}

func (m *ForceShardSyncMsg) Id() []byte {
	id := []byte("ForceShardSyncMsg")
	id = append(id, m.ShardId...)
	return append(id, m.Anchor.Signature...)
}

func (m *ForceShardSyncMsg) Code() uint64 {
	return ForceShardSyncMsgCode
}

func NewForceShardSyncMsg(shardId []byte, anchor *dto.Anchor) *ForceShardSyncMsg {
	return &ForceShardSyncMsg{
		ShardId: shardId,
		Anchor:  anchor,
	}
}

type TxShardChildRequestMsg struct {
	Hash [64]byte
}

func (m *TxShardChildRequestMsg) Id() []byte {
	return append([]byte("TxShardChildRequestMsg"), m.Hash[:]...)
}

func (m *TxShardChildRequestMsg) Code() uint64 {
	return TxShardChildRequestMsgCode
}

type TxShardChildResponseMsg struct {
	hash     [64]byte
	Bytes    []byte
	Children [][64]byte
}

func (m *TxShardChildResponseMsg) Id() []byte {
	return append([]byte("TxShardChildResponseMsg"), m.hash[:]...)
}

func (m *TxShardChildResponseMsg) Code() uint64 {
	return TxShardChildResponseMsgCode
}

func NewTxShardChildResponseMsg(tx dto.Transaction, children [][64]byte) *TxShardChildResponseMsg {
	if bytes, err := tx.Serialize(); err != nil {
		return nil
	} else {
		return &TxShardChildResponseMsg{
			hash:     tx.Id(),
			Bytes:    bytes,
			Children: children,
		}
	}
}

type ForceShardFlushMsg struct {
	hash  [64]byte
	Bytes []byte
}

func (m *ForceShardFlushMsg) Id() []byte {
	return append([]byte("ForceShardFlushMsg"), m.hash[:]...)
}

func (m *ForceShardFlushMsg) Code() uint64 {
	return ForceShardFlushMsgCode
}

func NewForceShardFlushMsg(tx dto.Transaction) *ForceShardFlushMsg {
	if bytes, err := tx.Serialize(); err != nil {
		return nil
	} else {
		return &ForceShardFlushMsg{
			hash:  tx.Id(),
			Bytes: bytes,
		}
	}
}
