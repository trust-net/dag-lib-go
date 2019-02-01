// Copyright 2018-2019 The trust-net Authors
// API DTOs for transaction Ops

package api

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"net/http"
)

// A request to submit a transaction
type SubmitRequest struct {
	// payload for transaction's operations
	Payload string `json:"payload"`
	// shard id for the transaction
	ShardId string `json:"shard_id"`
	// submitter's last transaction
	LastTx string `json:"last_tx"`
	// Submitter's public ID
	SubmitterId string `json:"submitter_id"`
	// submitter's transaction sequence
	SubmitterSeq uint64 `json:"submitter_seq"`
	// a padding to meet challenge for network's DoS protection
	Padding uint64 `json:"padding"`
	// signature of the transaction request's contents using submitter's private key
	Signature string `json:"signature"`

	txReq *dto.TxRequest
}

func (req *SubmitRequest) DltRequest() *dto.TxRequest {
	return req.txReq
}

func ParseSubmitRequest(r *http.Request) (*SubmitRequest, error) {
	req := &SubmitRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, fmt.Errorf("Malformed request: %s", err)
	}
	txReq := &dto.TxRequest{
		SubmitterSeq: req.SubmitterSeq,
		Padding: req.Padding,
	}

	if payload, _ := base64.StdEncoding.DecodeString(req.Payload); len(payload) == 0 {
		return nil, fmt.Errorf("malformed payload")
	} else {
		txReq.Payload = payload
	}
	if txReq.ShardId, _ = hex.DecodeString(req.ShardId); len(txReq.ShardId) == 0 {
		return nil, fmt.Errorf("invalid shard_id")
	}
	if bytes, _ := hex.DecodeString(req.LastTx); len(bytes) != 64 {
		return nil, fmt.Errorf("invalid shard_parent")
	} else {
		copy(txReq.LastTx[:], bytes)
	}
	if txReq.SubmitterId, _ = hex.DecodeString(req.SubmitterId); len(txReq.SubmitterId) == 0 {
		return nil, fmt.Errorf("invalid submitter_id")
	}
	if txReq.Signature, _ = base64.StdEncoding.DecodeString(req.Signature); len(txReq.Signature) == 0 {
		return nil, fmt.Errorf("invalid signature")
	}
	req.txReq = txReq
	return req, nil
}

//// A request to submit a transaction
//type SubmitRequest struct {
//	Anchor    *AnchorResponse `json:"anchor"`
//	Payload   string          `json:"payload"`
//	Signature string          `json:"tx_signature"`
//	tx        dto.Transaction
//}
//
//func (req *SubmitRequest) DltTransaction() dto.Transaction {
//	return req.tx
//}
//
//func ParseSubmitRequest(r *http.Request) (*SubmitRequest, error) {
//	req := &SubmitRequest{}
//	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
//		return nil, fmt.Errorf("Malformed request: %s", err)
//	}
//	// check for anchor in submission request
//	if req.Anchor == nil {
//		return nil, fmt.Errorf("missing anchor")
//	}
//	// construct DLT transaction
//	if anchor, err := req.Anchor.DltAnchor(); err != nil {
//		return nil, err
//	} else {
//		req.tx = dto.NewTransaction(anchor)
//	}
//	// copy payload and signature
//	if payload, _ := base64.StdEncoding.DecodeString(req.Payload); len(payload) == 0 {
//		return nil, fmt.Errorf("malformed payload")
//	} else {
//		req.tx.Self().Payload = payload
//	}
//	if signature, _ := base64.StdEncoding.DecodeString(req.Signature); len(signature) == 0 {
//		return nil, fmt.Errorf("malformed tx_signature")
//	} else {
//		req.tx.Self().Signature = signature
//	}
//	return req, nil
//}

// response to successful submission of a transaction
type SubmitResponse struct {
	TxId string `json:"tx_id"`
}

func NewSubmitResponse(tx dto.Transaction) *SubmitResponse {
	txId := tx.Id()
	res := &SubmitResponse{
		TxId: hex.EncodeToString(txId[:]),
	}
	return res
}
