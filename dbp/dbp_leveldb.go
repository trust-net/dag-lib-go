// Copyright 2019 The trust-net Authors
// DLT Stack's DB Provider implementation over leveldb
package dbp

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
)

var logger = log.NewLogger("dbpLevelDb")

// provide a of DBP implementtion based upon levelDB
func NewDbp(dirRoot string) (db.DbProvider, error) {
	// check for status of the specified directory
	if fs, err := os.Stat(dirRoot); err == nil {
		logger.Debug("%s exists: %s", dirRoot, fs.Mode().String())
		// check for write permission on directory
		if (fs.Mode() & 0x100) != 0x100 {
			logger.Error("Cannot write to %s", dirRoot)
			return nil, fmt.Errorf("directory not writable: %s", fs.Mode().String())
		}
	} else {
		// check the type of error
		if os.IsNotExist(err) {
			// try to create the directory (along with path)
			if err := createDir(dirRoot); err != nil {
				// issue with provided directory path
				logger.Error("Cannot create %s: %s", dirRoot, err)
				return nil, err
			}
		} else if os.IsPermission(err) {
			// we have permission issue with provided directory path
			logger.Error("Cannot access %s: %s", dirRoot, err)
			return nil, err
		}
	}
	logger.Debug("Created a DB Provider instance at directory root: %s", dirRoot)

	// Ensure we have some minimal caching and file guarantees
		cache := 16
		handles := 16
	ldb, err := leveldb.OpenFile(dirRoot, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2,
		WriteBuffer:            cache / 4, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		ldb, err = leveldb.RecoverFile(dirRoot, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}

	return &dbpLevelDb{
		dirRoot: dirRoot,
		ldb: ldb,
		repos:   make(map[string]*dbLevelDB),
	}, nil
}

func createDir(path string) error {
	return os.MkdirAll(path, os.ModeDir|os.ModePerm)
}

func removeDir(path string) error {
	return os.RemoveAll(path)
}

func makeReadOnly(path string) error {
	return os.Chmod(path, os.ModeDir)
}

func makeReadWrite(path string) error {
	return os.Chmod(path, os.ModeDir|os.ModePerm)
}

type dbpLevelDb struct {
	// directory root for each database to be provided
	dirRoot string
	// LevelDB instance
	ldb *leveldb.DB
	// open DB connections
	repos map[string]*dbLevelDB
}

func (dbp *dbpLevelDb) CloseAll() error {
	for _, db := range dbp.repos {
		db.Close()
	}
	// compact the DB
	logger.Debug("Compacting database ...")
	if err := dbp.ldb.CompactRange(util.Range{}); err != nil {
		logger.Error("Failed to compact db: %s", err)
		return err
	}
	logger.Debug("Compacting done.")
	logger.Debug("Closing database ...")
	defer logger.Debug("Close done.")
	return dbp.ldb.Close()
}

func (dbp *dbpLevelDb) DB(namespace string) db.Database {
	// check if DB connection already exists
	if repo, exists := dbp.repos[namespace]; exists {
		logger.Debug("re-using already open DB: %s", namespace)
		return repo
	}
//	// create a subdirectory for the namespace
//	if err := createDir(dbp.dirRoot + "/" + namespace); err != nil {
//		// issue with provided directory path
//		logger.Error("Cannot create %s: %s", dbp.dirRoot+"/"+namespace, err)
//		return nil
//	}
	if repo, err := newDbLevelDB(dbp, namespace); err != nil {
		logger.Error("Failed to instantiate namespace %s: %s", namespace, err)
		return nil
	} else {
		dbp.repos[namespace] = repo
		logger.Debug("opened database for namespace: %s", namespace)
		return repo
	}
}
