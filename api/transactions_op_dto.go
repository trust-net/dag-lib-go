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
	Anchor    *AnchorResponse `json:"anchor"`
	Payload   string          `json:"payload"`
	Signature string          `json:"tx_signature"`
	tx        dto.Transaction
}

func (req *SubmitRequest) DltTransaction() dto.Transaction {
	return req.tx
}

func ParseSubmitRequest(r *http.Request) (*SubmitRequest, error) {
	req := &SubmitRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, fmt.Errorf("Malformed request: %s", err)
	}
	// check for anchor in submission request
	if req.Anchor == nil {
		return nil, fmt.Errorf("missing anchor")
	}
	// construct DLT transaction
	if anchor, err := req.Anchor.DltAnchor(); err != nil {
		return nil, err
	} else {
		req.tx = dto.NewTransaction(anchor)
	}
	// copy payload and signature
	if payload, _ := base64.StdEncoding.DecodeString(req.Payload); len(payload) == 0 {
		return nil, fmt.Errorf("malformed payload")
	} else {
		req.tx.Self().Payload = payload
	}
	if signature, _ := base64.StdEncoding.DecodeString(req.Signature); len(signature) == 0 {
		return nil, fmt.Errorf("malformed tx_signature")
	} else {
		req.tx.Self().Signature = signature
	}
	return req, nil
}

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
