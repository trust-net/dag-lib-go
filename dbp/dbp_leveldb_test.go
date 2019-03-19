// Copyright 2019 The trust-net Authors
// Tests for DLT Stack's DB Provider implementation over leveldb
package dbp

import (
	"github.com/trust-net/dag-lib-go/log"
	"testing"
)

func cleanup(path string) {
	makeReadWrite(path)
	removeDir(path)
}

func Test_NewDbp_CreatePermissionCheck(t *testing.T) {
	log.SetLogLevel(log.NONE)
	createDir("tmp")
	makeReadOnly("tmp")
	defer cleanup("tmp")

	if _, err := NewDbp("tmp/that/cannot/be/created"); err == nil {
		t.Errorf("failed to check for inaccessible directory")
	}
}

func Test_NewDbp_CreateDirectory(t *testing.T) {
	log.SetLogLevel(log.NONE)
	// delete if directory already present
	dirPath := "tmp/that/is/not/present"
	cleanup(dirPath)
	defer cleanup("tmp")
	if dbp, err := NewDbp(dirPath); err != nil {
		t.Errorf("failed to create directory path: %s", err)
	} else if dbp.(*dbpLevelDb).dirRoot != dirPath {
		t.Errorf("directory root not initialized correctly")
	}
}

func Test_NewDbp_WritePermissionCheck(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp/not/writable"
	createDir(dirPath)
	makeReadOnly(dirPath)
	defer cleanup("tmp")
	if _, err := NewDbp(dirPath); err == nil {
		t.Errorf("failed to check for write permission on directory")
	}
}

func Test_NewDbp_DirectoryExists(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp/exists"
	createDir(dirPath)
	defer cleanup("tmp")
	if dbp, err := NewDbp(dirPath); err != nil {
		t.Errorf("failed to instantiate for existing directory: %s", err)
	} else if dbp.(*dbpLevelDb).dirRoot != dirPath {
		t.Errorf("directory root not initialized correctly")
	}
}

func Test_DB_DirectoryExists(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	createDir(dirPath + "/" + namespace)
	defer cleanup("tmp")
	dbp, _ := NewDbp(dirPath)
	if db := dbp.DB(namespace); db == nil {
		t.Errorf("failed to provide namespace for existing directory")
	}
}

func Test_DB_Reopen(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	defer cleanup("tmp")
	dbp, _ := NewDbp(dirPath)
	// create db
	db := dbp.DB(namespace)
	// close db
	db.Close()
	// re-open db that was already created
	if db := dbp.DB(namespace); db == nil {
		t.Errorf("failed to reopen db namespace")
	}
}

func Test_DB_OpenOpen(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	namespace := "test"
	defer cleanup("tmp")
	dbp, _ := NewDbp(dirPath)
	// create db
	dbp.DB(namespace)
	// re-open db that was already created
	if db := dbp.DB(namespace); db == nil {
		t.Errorf("failed to reopen db namespace")
	}
}

func Test_DB_Namepsaces(t *testing.T) {
	log.SetLogLevel(log.NONE)
	dirPath := "tmp"
	defer cleanup("tmp")
	namespace1 := "test-1"	
	namespace2 := "test-2"	
	// create two db's in different name spaces
	dbp, _ := NewDbp(dirPath)
	db1 := dbp.DB(namespace1)
	db2 := dbp.DB(namespace2)

	// put differnt values for some key in db's from different namespaces
	db1.Put([]byte("test-key-1"), []byte("test-value-1" + namespace1))
	db2.Put([]byte("test-key-1"), []byte("test-value-1" + namespace2))
	// put another key only in one of the name space
	db2.Put([]byte("test-key-2"), []byte("test-value-2"))
	
	// validate that both namespaces have their own different values for same common key
	if value, _ := db1.Get([]byte("test-key-1")); string(value)  != ("test-value-1" + namespace1) {
		t.Errorf("got unexpected value: %s", value)
	}
	if value, _ := db2.Get([]byte("test-key-1")); string(value)  != ("test-value-1" + namespace2) {
		t.Errorf("got unexpected value: %s", value)
	}
	
	// validate that only one namepsace has the other key
	if exists, _ := db1.Has([]byte("test-key-2")); exists {
		t.Errorf("got incorrect exists check: %v", exists)
	}
	if exists, _ := db2.Has([]byte("test-key-2")); !exists {
		t.Errorf("got incorrect exists check: %v", exists)
	}
}
