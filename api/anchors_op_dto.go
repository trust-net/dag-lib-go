// Copyright 2018-2019 The trust-net Authors
// API DTOs for anchors Ops

package api

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"net/http"
)

// A request to create a new anchor
type AnchorRequest struct {
	Submitter      string `json:"submitter,omitempty"`
	LastTx         string `json:"last_tx,omitempty"`
	NextSeq        uint64 `json:"next_seq"`
	submitterBytes []byte
	lastTxBytes    [64]byte
}

func ParseAnchorRequest(r *http.Request) (*AnchorRequest, error) {
	req := &AnchorRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, fmt.Errorf("Malformed request: %s", err)
	}
	// convert hex to bytes and validate parameters
	if req.submitterBytes, _ = hex.DecodeString(req.Submitter); len(req.submitterBytes) == 0 {
		return nil, fmt.Errorf("malformed submitter")
	}
	if lastTx, _ := hex.DecodeString(req.LastTx); len(lastTx) == 0 {
		return nil, fmt.Errorf("malformed last_tx")
	} else {
		copy(req.lastTxBytes[:], lastTx)
	}
	if req.NextSeq < 1 {
		return nil, fmt.Errorf("invalid next_seq")
	}
	return req, nil
}

func (r *AnchorRequest) SubmitterBytes() []byte {
	return r.submitterBytes
}

func (r *AnchorRequest) LastTxBytes() [64]byte {
	return r.lastTxBytes
}

// an API specs compatible anchor response
type AnchorResponse struct {
	// transaction approver application instance node ID
	NodeId string `json:"node_id"`
	// transaction approver application's shard ID
	ShardId string `json:"shard_id"`
	// sequence of this transaction within the shard
	ShardSeq uint64 `json:"shard_seq"`
	// weight of this transaction withing shard DAG (sum of all ancestor's weight + 1)
	Weight uint64 `json:"weight"`
	// parent transaction within the shard
	ShardParent string `json:"shard_parent"`
	// uncle transactions within the shard
	ShardUncles []string `json:"shard_uncles"`
	// transaction submitter's public ID
	Submitter string `json:"submitter"`
	// submitter's last transaction ID
	SubmitterLastTx string `json:"last_tx"`
	// submitter's transaction sequence number
	SubmitterSeq uint64 `json:"submitter_seq"`
	// anchor signature from DLT stack
	Signature string `json:"signature"`
}

func NewAnchorResponse(a *dto.Anchor) *AnchorResponse {
	res := &AnchorResponse{
		NodeId:          hex.EncodeToString(a.NodeId),
		ShardId:         hex.EncodeToString(a.ShardId),
		ShardSeq:        a.ShardSeq,
		Weight:          a.Weight,
		ShardParent:     hex.EncodeToString(a.ShardParent[:]),
		ShardUncles:     make([]string, len(a.ShardUncles)),
		Submitter:       hex.EncodeToString(a.Submitter),
		SubmitterLastTx: hex.EncodeToString(a.SubmitterLastTx[:]),
		SubmitterSeq:    a.SubmitterSeq,
		Signature:       base64.StdEncoding.EncodeToString(a.Signature),
	}
	for i, uncle := range a.ShardUncles {
		res.ShardUncles[i] = hex.EncodeToString(uncle[:])
	}
	return res
}
