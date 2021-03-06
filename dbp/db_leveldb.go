// Copyright 2019 The trust-net Authors
// DLT Stack's Database implementation over leveldb
package dbp

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/trust-net/dag-lib-go/log"
)

type dbLevelDB struct {
	// namespace for the DB
	namespace string
	// LevelDB instance
	ldb *leveldb.DB
	// logger
	logger log.Logger
	// connection status
	isOpen bool
}

func newDbLevelDB(namespace string, path string, cache int, handles int) (*dbLevelDB, error) {
	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	ldb, err := leveldb.OpenFile(path, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2,
		WriteBuffer:            cache / 4, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		ldb, err = leveldb.RecoverFile(path, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}

	db := &dbLevelDB{
		ldb:       ldb,
		namespace: namespace,
		logger:    log.NewLogger("db-" + namespace),
		isOpen:    true,
	}
	return db, nil
}

func (db *dbLevelDB) GetAll() [][]byte {
	// get an iterator over DB
	it := db.ldb.NewIterator(nil, nil)
	if it == nil || !it.First() {
		db.logger.Debug("empty iterator from DB")
		return nil
	} else {
		defer it.Release()
	}

	// loop through iterator and add to values
	values := make([][]byte, 0)
	done := false
	for !done {
		// copy over bytes, since iterator re-uses the existing slice, and append is copying reference only
		value := make([]byte, len(it.Value()))
		copy(value, it.Value())
		values = append(values, value)
		done = !it.Next()
	}
	db.logger.Error("getall has %d elements", len(values))
	return values
}

func (db *dbLevelDB) Name() string {
	return db.namespace
}

func (db *dbLevelDB) Put(key []byte, value []byte) error {
	return db.ldb.Put(key, value, nil)
}

func (db *dbLevelDB) Get(key []byte) ([]byte, error) {
	return db.ldb.Get(key, nil)
}

func (db *dbLevelDB) Has(key []byte) (bool, error) {
	return db.ldb.Has(key, nil)
}

func (db *dbLevelDB) Delete(key []byte) error {
	return db.ldb.Delete(key, nil)
}

func (db *dbLevelDB) Close() error {
	db.isOpen = false
	// compact the DB
	db.logger.Debug("Compacting database ...")
	if err := db.ldb.CompactRange(util.Range{}); err != nil {
		db.logger.Error("Failed to compact db: %s", err)
		return err
	}
	db.logger.Debug("Compacting done.")
	db.logger.Debug("Closing database ...")
	defer db.logger.Debug("Close done.")
	return db.ldb.Close()
}

func (db *dbLevelDB) Drop() error {
	// get an iterator over DB
	it := db.ldb.NewIterator(nil, nil)
	if it == nil || !it.First() {
		db.logger.Debug("empty iterator from DB")
		return nil
	} else {
		defer it.Release()
	}

	// loop through iterator and delete keys
	done := false
	count := 0
	for !done {
		// copy over bytes, since iterator re-uses the existing slice, and append is copying reference only
		if err := db.Delete(it.Key()); err != nil {
			return err
		}
		count += 1
		done = !it.Next()
	}
	db.logger.Debug("dropped %d elements", count)
	return nil
}
