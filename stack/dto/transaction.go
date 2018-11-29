// Copyright 2018 The trust-net Authors
// Common DTO types used throughout DLT stack
package dto

import (
	"github.com/trust-net/go-trust-net/common"
)

// transaction message
type Transaction struct {
	// serialized transaction payload
	Payload []byte
	// transaction signature
	Signature []byte
	// transaction approver application instance ID
	AppId []byte
	// transaction approver application's shard ID
	ShardId []byte
	// transaction submitter's public ID
	Submitter []byte
}

func (tx *Transaction) Serialize() ([]byte, error) {
	return common.Serialize(tx)
}

func (tx *Transaction) DeSerialize(data []byte) (error) {
	if err := common.Deserialize(data, tx); err != nil {
		return err
	}
	return nil
}