// Copyright 2019 The trust-net Authors
// Tests for DLT Stack's Database implementation over leveldb
package dbp

import (
	"github.com/trust-net/dag-lib-go/log"
	"testing"
)

func Test_NewDbLevelDB_OpenError(t *testing.T) {
	log.SetLogLevel(log.NONE)
	namespace := "test"
	dirPath := "tmp"
	cleanup(dirPath)
	defer cleanup("tmp")
	// create db provider instance
	dbp, _ := NewDbp(dirPath)

	// make directory read only, so that db creation fails
	makeReadOnly(dirPath)

	if db := dbp.DB(namespace); db != nil {
		t.Skipf("not checking for inaccessible directory during DB create")
	}
}

func Test_NewDbLevelDB_OpenSuccess(t *testing.T) {
	log.SetLogLevel(log.NONE)
	namespace := "test"
	dirPath := "tmp"
	cleanup(dirPath)
	defer cleanup("tmp")
	// create db provider instance
	dbp, _ := NewDbp(dirPath)

	if db := dbp.DB(namespace); db == nil {
		t.Errorf("failed to create db")
	}
}

func Test_Db_Name(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// make sure namespace is correct
	if db.Name() != namespace {
		t.Errorf("incorrect namespace")
	}
}

func Test_Db_Put(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// put some value
	if err := db.Put([]byte("test-key"), []byte("test-value")); err != nil {
		t.Errorf("failed to put into db: %s", err)
	}
}

func Test_Db_PutAfterPut(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// put some value
	db.Put([]byte("test-key"), []byte("test-value-old"))
	// now put a new value for same key
	db.Put([]byte("test-key"), []byte("test-value-new"))

	// get same key, it should get new value
	if value, err := db.Get([]byte("test-key")); err != nil {
		t.Errorf("failed to get from db: %s", err)
	} else if string(value) != "test-value-new" {
		t.Errorf("got incorect value: %s", value)
	}
}

func Test_Db_GetAfterPut(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// put some value
	db.Put([]byte("test-key"), []byte("test-value"))

	// get same value
	if value, err := db.Get([]byte("test-key")); err != nil {
		t.Errorf("failed to get from db: %s", err)
	} else if string(value) != "test-value" {
		t.Errorf("got incorect value: %s", value)
	}
}

func Test_Db_GetNotExisting(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// try to get some key that was never put
	if value, err := db.Get([]byte("test-key")); err == nil {
		t.Errorf("got unexpected value: %s", value)
	}
}

func Test_Db_HasAfterPut(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// put some value
	db.Put([]byte("test-key"), []byte("test-value"))

	// check if key exists
	if exists, err := db.Has([]byte("test-key")); err != nil {
		t.Errorf("failed to check if exists in db: %s", err)
	} else if !exists {
		t.Errorf("got incorrect exists check: %v", exists)
	}
}

func Test_Db_HasNotExisting(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// try to check if key exists that was never put
	if exists, _ := db.Has([]byte("test-key")); exists {
		t.Errorf("got incorect exists check: %v", exists)
	}
}

func Test_Db_DeleteAfterPut(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// put some value
	db.Put([]byte("test-key"), []byte("test-value"))

	// delete same value
	if err := db.Delete([]byte("test-key")); err != nil {
		t.Errorf("failed to delete from db: %s", err)
	} else if exists, _ := db.Has([]byte("test-key")); exists {
		t.Errorf("got incorect exists after delete: %s", exists)
	}
}

func Test_Db_DeleteNotExisting(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// try to delete some key that was never put
	if err := db.Delete([]byte("test-key")); err != nil {
		t.Errorf("error upon non existing key delete from db: %s", err)
	}
}

func Test_Db_CloseAfterOpen(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// close db
	if err := db.Close(); err != nil {
		t.Errorf("error upon closing db: %s", err)
	}
}

func Test_Db_CloseAfterClose(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// close db
	db.Close()

	// re-attempt to close the closed db
	if err := db.Close(); err == nil {
		t.Skipf("did not detect already closed db")
	}
}

func Test_Db_GetAll(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)
	// create another db for different name space
	db2 := dbp.DB(namespace+"-2")

	// put some value
	db.Put([]byte("test-key-1"), []byte("test-value-1"))
	db.Put([]byte("test-key-2"), []byte("test-value-2"))
	db.Put([]byte("test-key-3"), []byte("test-value-3"))
	
	// put values in different namespace db
	db2.Put([]byte("test-key-4"), []byte("test-value-4"))

	// get all values from first DB
	values := db.GetAll()
	if len(values) != 3 {
		t.Errorf("get all has incorrect number of items: %d", len(values))
	} else {
		found := map[string]bool{
			"test-value-1": false,
			"test-value-2": false,
			"test-value-3": false,
		}
		for _, value := range values {
			logger.Debug("got value %s", value)
			found[string(value)] = true
		}
		for v, seen := range found {
			if !seen {
				t.Errorf("did not find %s in get all", v)
			}
		}
	}
}

func Test_Db_Drop(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	cleanup(dirPath)
	defer cleanup(dirPath)

	// create a db
	dbp, _ := NewDbp(dirPath)
	db := dbp.DB(namespace)

	// put some value
	db.Put([]byte("test-key-1"), []byte("test-value-1"))
	db.Put([]byte("test-key-2"), []byte("test-value-2"))
	db.Put([]byte("test-key-3"), []byte("test-value-3"))

	// drop the table
	if err := db.Drop(); err != nil {
		t.Errorf("failed to drop table: %s", err)
	}

	// get all values
	values := db.GetAll()
	if len(values) != 0 {
		t.Errorf("did not expect any keys")
	} else {
		if exists, _ := db.Has([]byte("test-key-1")); exists {
			t.Errorf("got incorect exists after drop: %v", exists)
		}
		if exists, _ := db.Has([]byte("test-key-2")); exists {
			t.Errorf("got incorect exists after drop: %v", exists)
		}
		if exists, _ := db.Has([]byte("test-key-3")); exists {
			t.Errorf("got incorect exists after drop: %v", exists)
		}
	}
}
