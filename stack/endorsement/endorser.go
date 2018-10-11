// Copyright 2018 The trust-net Authors
// Endorsement Layer interface and implementation for DLT Statck
package endorsement

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
)

type Endorser interface {
	// TBD
}

type endorser struct {
	// TBD
}

func NewEndorser(db db.Database) (*endorser, error) {
	return nil, errors.New("not yet implemented")
}