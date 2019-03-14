// Copyright 2018 The trust-net Authors
// In memory implementation of DB interface for testing purpose
package db

import (
	"errors"
	"sync"
	"github.com/trust-net/dag-lib-go/log"
)

// in memory implementation of database (for testing etc.)
type inMemDb struct {
	mdb  map[string][]byte
	lock sync.RWMutex
	name string
	isOpen bool
	logger log.Logger
}

func NewInMemDatabase(name string) *inMemDb {
	return &inMemDb{
		mdb:  make(map[string][]byte),
		name: name,
		isOpen: true,
		logger: log.NewLogger("inMemDb-" + name),
	}
}

type inMemDbProvider struct {
	repos map[string]*inMemDb
	logger log.Logger
}

func NewInMemDbProvider() *inMemDbProvider {
	return &inMemDbProvider{
		repos: make(map[string]*inMemDb),
		logger: log.NewLogger("inMemDbProvider"),
	}
}

func (p *inMemDbProvider) DB(ns string) Database {
	if db, exists := p.repos[ns]; exists {
		if db.isOpen {
			p.logger.Error("DB already open: %s", ns)
			return nil
		} else {
			p.logger.Debug("DB re-opened: %s", ns)
			db.isOpen = true
			return db
		}
	} else {
		db = NewInMemDatabase(ns)
		p.repos[ns] = db
		p.logger.Debug("DB opened for: %s", ns)
		return db
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

func (db *inMemDb) GetAll() [][]byte {
	db.lock.Lock()
	defer db.lock.Unlock()
	values := make([][]byte, len(db.mdb))
	i := 0
	for _, value := range db.mdb {
		values[i] = value
		i += 1
	}
	return values
}

func (db *inMemDb) Flush() {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.mdb = make(map[string][]byte)
}

func (db *inMemDb) Has(key []byte) (bool, error) {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, ok := db.mdb[string(key)]
	return ok, nil
}

func (db *inMemDb) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.mdb, string(key))
	return nil
}

func (db *inMemDb) Close() error {
	db.isOpen = false
	db.logger.Debug("Closed DB: %s", db.name)
	return nil
}

func (db *inMemDb) Name() string {
	return db.name
}
