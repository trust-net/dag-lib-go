// Copyright 2018-2019 The trust-net Authors
// Common DTO types used throughout DLT stack
package dto

import (
	"crypto/sha512"
	"github.com/trust-net/dag-lib-go/common"
)

type Transaction interface {
	Id() [64]byte
	Serialize() ([]byte, error)
	DeSerialize(data []byte) error
	Anchor() *Anchor
	Request() *TxRequest
	Self() *transaction
}

// transaction message
type transaction struct {
	// id of the transaction created from its hash
	id     [64]byte
	idDone bool
	// transaction request from submitter
	TxRequest *TxRequest
	// transaction anchor from DLT stack
	TxAnchor *Anchor
}

// compute SHA512 hash or return from cache
func (tx *transaction) Id() [64]byte {
	if tx.idDone {
		return tx.id
	}
	data := make([]byte, 0, 128)
	// signature should be sufficient to capture payload and submitter ID
	data = append(data, tx.TxRequest.Signature...)
	// append anchor's signature
	data = append(data, tx.TxAnchor.Signature...)
	tx.id = sha512.Sum512(data)
	tx.idDone = true
	return tx.id
}

// serialize transaction for local DB storage, should not be used to transmit bytes over network
func (tx *transaction) Serialize() ([]byte, error) {
	return common.Serialize(tx)
}

// de-serialize transaction from local DB storage, should not be used to de-serialize from network bytes
// ###########################################################
// TBD: need to change dto.Transaction from interface to concrete type, so that p2p layer can do
// network transmission using rlp encoding and do not require this de-serialize method
// ###########################################################
func (tx *transaction) DeSerialize(data []byte) error {
	if err := common.Deserialize(data, tx); err != nil {
		return err
	}
	return nil
}

func (tx *transaction) Anchor() *Anchor {
	return tx.TxAnchor
}

func (tx *transaction) Request() *TxRequest {
	return tx.TxRequest
}

func (tx *transaction) Self() *transaction {
	return tx
}

// make sure any Transaction can only be created with a request and anchor
func NewTransaction(r *TxRequest, a *Anchor) *transaction {
	if r == nil || a == nil {
		return nil
	}
	return &transaction{
		TxRequest: r,
		TxAnchor:  a,
	}
}
