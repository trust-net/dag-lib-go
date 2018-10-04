// Copyright 2018 The trust-net Authors
// Sharding Layer interface and implementation for DLT Statck
package sharding

import (
	"errors"
	"github.com/trust-net/dag-lib-go/db"
)

type Sharder interface {
	// TBD
}

type sharder struct {
	// TBD
}

func NewSharder(db db.Database) (*sharder, error) {
	return nil, errors.New("not yet implemented")
}