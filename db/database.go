// Copyright 2018 The trust-net Authors
// A DB interface that application needs to provide to DLT stack
package db

import (

)

type Database interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close() error
}
