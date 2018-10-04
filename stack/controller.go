// Copyright 2018 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
)

type DLT interface {
	// TBD
}

type dlt struct {
	// TBD
}

func NewDltStack(db db.Database) (*dlt, error) {
	return nil, errors.New("not yet implemented")
}