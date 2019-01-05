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

func TestPut(t *testing.T) {
	s := testWorldState()

	if err := s.Put(&Resource{}); err != nil {
		t.Errorf("Failed to put: %s", err)
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
