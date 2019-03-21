// Copyright 2018 The trust-net Authors
// A DB interface that application needs to provide to DLT stack
package db

import ()

type Database interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	GetAll() [][]byte
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close() error
	Name() string
	Drop() error
}

type DbProvider interface {
	DB(namespace string) Database
	CloseAll() error
}
