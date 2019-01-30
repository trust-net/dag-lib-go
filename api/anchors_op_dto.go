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
	// 130 char hex encoded public id/key of the node issuing this anchor
	NodeId string `json:"node_id"`
	// a hex encoded string uniquely identifying the shard of the application
	ShardId string `json:"shard_id"`
	// 64 bit unsigned integer value for sequence of this anchor in the shard DAG
	ShardSeq uint64 `json:"shard_seq"`
	// 64 bit unsigned integer value for weight of this anchor in the shard DAG (sum of all ancestor's weight + 1)
	Weight uint64 `json:"weight"`
	// 128 char hex encoded id of the parent tip for this anchor within shard DAG
	ShardParent string `json:"shard_parent"`
	// list of 128 char hex encoded ids of other tips for this anchor within shard DAG
	ShardUncles []string `json:"shard_uncles"`
	// 130 char hex encoded public id of the submitter
	Submitter string `json:"submitter"`
	// 128 char hex encoded id of the last transaction from submitter
	SubmitterLastTx string `json:"last_tx"`
	// 64 bit unsigned integer value for anchor's transaction sequence from submitter
	SubmitterSeq uint64 `json:"submitter_seq"`
	// a base64 encoded ECDSA secpk256 signature using private key of issuing node
	Signature string `json:"anchor_signature"`
}

func (r *AnchorResponse) DltAnchor() (*dto.Anchor, error) {
	a := &dto.Anchor{
		ShardSeq:     r.ShardSeq,
		Weight:       r.Weight,
		SubmitterSeq: r.SubmitterSeq,
		ShardUncles:  make([][64]byte, len(r.ShardUncles)),
	}
	if a.NodeId, _ = hex.DecodeString(r.NodeId); len(a.NodeId) != 65 {
		return nil, fmt.Errorf("invalid node_id")
	}
	if a.ShardId, _ = hex.DecodeString(r.ShardId); len(a.ShardId) == 0 {
		return nil, fmt.Errorf("invalid shard_id")
	}
	if bytes, _ := hex.DecodeString(r.ShardParent); len(bytes) != 64 {
		return nil, fmt.Errorf("invalid shard_parent")
	} else {
		copy(a.ShardParent[:], bytes)
	}
	if a.Submitter, _ = hex.DecodeString(r.Submitter); len(a.Submitter) != 65 {
		return nil, fmt.Errorf("invalid submitter")
	}
	if bytes, _ := hex.DecodeString(r.SubmitterLastTx); len(bytes) != 64 {
		return nil, fmt.Errorf("invalid last_tx")
	} else {
		copy(a.SubmitterLastTx[:], bytes)
	}
	if a.Signature, _ = base64.StdEncoding.DecodeString(r.Signature); len(a.Signature) == 0 {
		return nil, fmt.Errorf("invalid anchor_signature")
	}
	for i, uncle := range r.ShardUncles {
		if bytes, _ := hex.DecodeString(uncle); len(bytes) != 64 {
			return nil, fmt.Errorf("invalid shard_uncle")
		} else {
			copy(a.ShardUncles[i][:], bytes)
		}
	}
	return a, nil
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
