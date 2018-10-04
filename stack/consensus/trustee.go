// Copyright 2018 The trust-net Authors
// Consensus Layer interface and implementation for DLT Statck
package consensus

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
)

type Trustee interface {
	// TBD
}

type trustee struct {
	// TBD
}

func NewTrustee(db db.Database) (*trustee, error) {
	return nil, errors.New("not yet implemented")
}