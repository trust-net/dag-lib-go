package state

import (
	"github.com/trust-net/dag-lib-go/db"
	"testing"
)

func testWorldState() *worldState {
	s, _ := NewWorldState(db.NewInMemDbProvider(), []byte("test shard"))
	return s
}

func TestInitialization(t *testing.T) {
	var s State
	var err error
	s = testWorldState()
	if err != nil || s == nil {
		t.Errorf("Failed to create instance: %s", err)
	}
}

func TestPutToCache(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	value := "test data"
	r := &Resource{
		Key:   key,
		Owner: []byte("test owner"),
		Value: []byte(value),
	}

	// put test resource into world state
	if err := s.Put(r); err != nil {
		t.Errorf("Failed to put resource: %s", err)
	}

	// put should not have been made to DB
	if _, err := s.stateDb.Get(key); err == nil {
		t.Errorf("did not expect value in DB")
	}
}

func TestDeleteToCache(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	value := "test data"
	r := &Resource{
		Key:   key,
		Owner: []byte("test owner"),
		Value: []byte(value),
	}

	// place the resource into DB
	data, _ := r.Serialize()
	s.stateDb.Put(key, data)

	// delte resource from world state
	if err := s.Delete(key); err != nil {
		t.Errorf("Failed to delete resource: %s", err)
	}

	// delete should not have been made to DB
	if data, err := s.stateDb.Get(key); err != nil || len(data) == 0 {
		t.Errorf("did not expect delete from DB")
	}
}

func TestGetAfterPut(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	value := "test data"
	s.Put(&Resource{
		Key:   key,
		Owner: []byte("test owner"),
		Value: []byte(value),
	})

	if r, err := s.Get(key); err != nil {
		t.Errorf("Failed to get: %s", err)
	} else if string(r.Value) != value {
		t.Errorf("Incorrect value: %s, expected: %s", r.Value, value)
	}
}

func TestGetNoPut(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	if r, _ := s.Get(key); r != nil {
		t.Errorf("Did not expect to get: %s", key)
	}
}

func TestGetAfterDelete(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	value := "test data"
	s.Put(&Resource{
		Key:   key,
		Owner: []byte("test owner"),
		Value: []byte(value),
	})

	s.Delete(key)

	if r, _ := s.Get(key); r != nil {
		t.Errorf("Did not expect to get: %s", key)
	}
}

func TestGetAfterReset(t *testing.T) {
	s := testWorldState()
	s.Put(&Resource{
		Key:   []byte("key1"),
		Owner: []byte("test owner 1"),
		Value: []byte("test data 1"),
	})
	s.Put(&Resource{
		Key:   []byte("key2"),
		Owner: []byte("test owner 2"),
		Value: []byte("test data 2"),
	})

	if err := s.Reset(); err != nil {
		t.Errorf("Failed to reset: %s", err)
	}

	if r, _ := s.Get([]byte("key1")); r != nil {
		t.Errorf("Did not expect to get key1")
	}
	if r, _ := s.Get([]byte("key2")); r != nil {
		t.Errorf("Did not expect to get key2")
	}
}

func TestGetFromCacheHit(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	value := "test data"
	s.cache["key1"] = &Resource{
		Key:   key,
		Owner: []byte("test owner"),
		Value: []byte(value),
	}

	if _, err := s.Get(key); err != nil {
		t.Errorf("Failed to get: %s", err)
	} else if _, err := s.stateDb.Get(key); err == nil {
		t.Errorf("did not expect value in DB")
	}
}

func TestGetFromCacheMiss(t *testing.T) {
	s := testWorldState()
	key := []byte("key1")
	value := "test data"
	// place a resource in DB (and not in cache)
	r := &Resource{
		Key:   key,
		Owner: []byte("test owner"),
		Value: []byte(value),
	}
	data, _ := r.Serialize()
	s.stateDb.Put(key, data)

	// get should be able fetch resource from DB, and update cache
	if _, err := s.Get(key); err != nil {
		t.Errorf("Failed to get: %s", err)
	} else if _, found := s.cache[string(key)]; !found {
		t.Errorf("did not find value in cache after get")
	}
}

func TestPersistToDb(t *testing.T) {
	s := testWorldState()
	// add a resource directly into cache
	s.cache["key1"] = &Resource{
		Key:   []byte("key1"),
		Owner: []byte("test owner 1"),
		Value: []byte("test data 1"),
	}
	// add another resource via put
	s.Put(&Resource{
		Key:   []byte("key2"),
		Owner: []byte("test owner 2"),
		Value: []byte("test data 2"),
	})
	// do a delete on third resource placed in DB
	r := &Resource{
		Key:   []byte("key3"),
		Owner: []byte("test owner 3"),
		Value: []byte("test data 3"),
	}
	data, _ := r.Serialize()
	s.stateDb.Put(r.Key, data)
	s.Delete([]byte("key3"))

	// now persist
	if err := s.Persist(); err != nil {
		t.Errorf("Failed to persist: %s", err)
	} else {
		// validate updates to DB
		if _, err := s.stateDb.Get([]byte("key1")); err != nil {
			t.Errorf("Did not find directly placed key1 in DB")
		}
		if _, err := s.stateDb.Get([]byte("key2")); err != nil {
			t.Errorf("Did not find updated key2 in DB")
		}
		if _, err := s.stateDb.Get([]byte("key3")); err == nil {
			t.Errorf("Should not find deleted key3 in DB")
		}
	}
}
