// Copyright 2019 The trust-net Authors
// DLT Stack's DB Provider implementation over leveldb
package dbp

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"github.com/trust-net/dag-lib-go/log"
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
	return &dbpLevelDb{
		dirRoot: dirRoot,
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
	// open DB connections
	repos map[string]*dbLevelDB
}

func (dbp *dbpLevelDb) CloseAll() error {
	for _, db := range dbp.repos {
		db.Close()
	}
	return nil
}

func (dbp *dbpLevelDb) DB(namespace string) db.Database {
	// check if DB connection already exists
	if repo, exists := dbp.repos[namespace]; exists {
		if repo.isOpen {
			logger.Debug("re-using already open DB: %s", namespace)
			return repo
		} else {
			logger.Debug("DB is closed: %s", namespace)
		}
	}
	// create a subdirectory for the namespace
	if err := createDir(dbp.dirRoot + "/" + namespace); err != nil {
		// issue with provided directory path
		logger.Error("Cannot create %s: %s", dbp.dirRoot+"/"+namespace, err)
		return nil
	}
	if repo, err := newDbLevelDB(namespace, dbp.dirRoot+"/"+namespace, 16, 16); err != nil {
		logger.Error("Failed to instantiate namespace %s: %s", namespace, err)
		return nil
	} else {
		dbp.repos[namespace] = repo
		logger.Debug("opened database for namespace: %s", namespace)
		return repo
	}
}
