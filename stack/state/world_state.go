// Copyright 2019 The trust-net Authors
// World State interface to manage resources for an APP using DLT stack
package state

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"sync"
)

type State interface {
// used to check if a transaction is already seen by the shard, so as to skip duplicates
// also, marks the transaction as seen for any future reference
	Seen(txId []byte) bool
	Get(key []byte) (*Resource, error)
	Put(r *Resource) error
	Delete(key []byte) error
	Persist() error
	Reset() error
	Close() error
}

type worldState struct {
	stateDb db.Database
	seenTxDb db.Database
	// in mem cache for resource updates, until transaction is completely accepted and persisted
	cache map[string]*Resource
	// TBD: following should be redundant, since we are locking at sharding layer before passing this reference
	// to app for transaction processing -- but then we never know how app is using it. Also, protects during any
	// reads happening outside of transaction processing
	lock sync.RWMutex
}

func (s *worldState) Get(key []byte) (*Resource, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// first look into cache
	if r, found := s.cache[string(key)]; !found {
		// not found, so read from DB and cache
		if data, err := s.stateDb.Get(key); err == nil {
			r := &Resource{}
			if err = r.DeSerialize(data); err == nil {
				s.cache[string(key)] = r
				return r, nil
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		return r, nil
	}
}

// delete will put nil as value
func (s *worldState) Delete(key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cache[string(key)] = nil
	return nil
}

// used to check if a transaction is already seen by the shard, so as to skip duplicates
// also, marks the transaction as seen for any future reference
func (s *worldState) Seen(txId []byte) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	isSeen, _ := s.seenTxDb.Has(txId)
	if !isSeen {
		s.seenTxDb.Put(txId, []byte{})
	}
	return isSeen
	
}

func (s *worldState) Put(r *Resource) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if r == nil || len(r.Key) == 0 {
		return fmt.Errorf("nil resource or key")
	}
	s.cache[string(r.Key)] = r
	return nil
}

func (s *worldState) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.seenTxDb.Close()
	return s.stateDb.Close()
}
func (s *worldState) Persist() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for k, r := range s.cache {
		if r == nil {
			// delete from DB
			if err := s.stateDb.Delete([]byte(k)); err != nil {
				return err
			}
		} else {
			// serialize resource
			if data, err := r.Serialize(); err != nil {
				return err
			} else {
				// update in DB
				if err := s.stateDb.Put(r.Key, data); err != nil {
					return err
				}
			}
		}
	}
	// flush the cache
	s.cache = make(map[string]*Resource)
	return nil
}

func (s *worldState) Reset() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cache = make(map[string]*Resource)
	for _, data := range s.stateDb.GetAll() {
		r := &Resource{}
		var err error
		if err = r.DeSerialize(data); err == nil {
			err = s.stateDb.Delete(r.Key)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func NewWorldState(dbp db.DbProvider, shardId []byte) (*worldState, error) {
	if stateDb := dbp.DB("Shard-World-State-" + string(shardId)); stateDb != nil {
		if seenTxDb := dbp.DB("Shard-Seen-Tx-" + string(shardId)); seenTxDb != nil {
			return &worldState{
				stateDb: stateDb,
				seenTxDb: seenTxDb,
				cache:   make(map[string]*Resource),
			}, nil
		}
	}
	return nil, fmt.Errorf("could not instantiate DB")
}
