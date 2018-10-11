// Copyright 2018 The trust-net Authors
// In memory implementation of DB interface for testing purpose
package db

import (
	"errors"
	"sync"
)
// in memory implementation of database (for testing etc.)
type inMemDb struct {
	mdb map[string][]byte
	lock sync.RWMutex
}

func NewInMemDatabase() *inMemDb {
	return &inMemDb{
		mdb: make(map[string][]byte),
	}
}

func (db *inMemDb) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.mdb[string(key)] = value
	return nil
}

func (db *inMemDb) Get(key []byte) ([]byte, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	if data, ok := db.mdb[string(key)]; !ok {
		return data, errors.New("not found")
	} else {
		return data, nil
	}
}

func (db *inMemDb) Has(key []byte) (bool, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, ok := db.mdb[string(key)]
	return  ok, nil
}

func (db *inMemDb) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.mdb, string(key))
	return nil
}

func (db *inMemDb) Close() error{
	return nil
}