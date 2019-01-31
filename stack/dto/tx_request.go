package dto

import (
	"github.com/trust-net/dag-lib-go/common"
)

type TxRequest struct {
	// payload for transaction's operations
	Payload []byte
	// shard id for the transaction
	ShardId []byte
	// submitter's last transaction
	LastTx [64]byte
	// Submitter's public ID
	SubmitterId []byte
	// submitter's transaction sequence
	SubmitterSeq uint64
	// a padding to meet challenge for network's DoS protection
	Padding uint64
	// signature of the transaction request's contents using submitter's private key
	Signature []byte
}

// we want to make sure we always create byte array for signature in a language indpendent order
func (r *TxRequest) Bytes() []byte {
	payload := make([]byte, 0, len(r.Payload)+len(r.ShardId)+144)
	payload = append(payload, r.Payload...)
	payload = append(payload, r.ShardId...)
	payload = append(payload, r.LastTx[:]...)
	payload = append(payload, r.SubmitterId...)
	payload = append(payload, common.Uint64ToBytes(r.SubmitterSeq)...)
	payload = append(payload, common.Uint64ToBytes(r.Padding)...)
	return payload
}
