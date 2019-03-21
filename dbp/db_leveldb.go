// Copyright 2019 The trust-net Authors
// DLT Stack's Database implementation over leveldb
package dbp

import (
	"github.com/trust-net/dag-lib-go/log"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type dbLevelDB struct {
	// db provider reference
	dbp *dbpLevelDb
	// prefix for the DB
	prefix []byte
	// namspace for the DB
	namespace string
//	// LevelDB instance
//	ldb *leveldb.DB
	// logger
	logger log.Logger
//	// connection status
//	isOpen bool
}

func newDbLevelDB(dbp *dbpLevelDb, namespace string) (*dbLevelDB, error) {
	db := &dbLevelDB{
		dbp: dbp,
		namespace: namespace,
		prefix: []byte("start-" + namespace + "-end:"),
		logger:    log.NewLogger("db-" + namespace),
	}
	return db, nil
}

func (db *dbLevelDB) GetAll() [][]byte {
	// get an iterator over DB
	it := db.dbp.ldb.NewIterator(util.BytesPrefix([]byte(db.prefix)), nil)
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
	db.logger.Debug("getall has %d elements", len(values))
	return values
}

func (db *dbLevelDB) Name() string {
	return db.namespace
}

func (db *dbLevelDB) Put(key []byte, value []byte) error {
	return db.dbp.ldb.Put(append(db.prefix, key...), value, nil)
}

func (db *dbLevelDB) Get(key []byte) ([]byte, error) {
	return db.dbp.ldb.Get(append(db.prefix, key...), nil)
}

func (db *dbLevelDB) Has(key []byte) (bool, error) {
	return db.dbp.ldb.Has(append(db.prefix, key...), nil)
}

func (db *dbLevelDB) Delete(key []byte) error {
	return db.dbp.ldb.Delete(append(db.prefix, key...), nil)
}

func (db *dbLevelDB) Close() error {
	db.dbp = nil
	return nil
}

func (db *dbLevelDB) Drop() error {
	// get an iterator over DB
	it := db.dbp.ldb.NewIterator(util.BytesPrefix([]byte(db.prefix)), nil)
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
		if err := db.dbp.ldb.Delete(it.Key(), nil); err != nil {
			return err
		}
		count += 1
		done = !it.Next()
	}
	db.logger.Debug("dropped %d elements", count)
	return nil
}
