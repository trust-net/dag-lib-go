// Copyright 2019 The trust-net Authors
// World State interface to manage resources for an APP using DLT stack
package state

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/db"
	"sync"
)

type State interface {
	Get(key []byte) (*Resource, error)
	Put(r *Resource) error
	Delete(key []byte) error
}

type worldState struct {
	stateDb db.Database
	lock    sync.RWMutex
}

func (s *worldState) Get(key []byte) (*Resource, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if data, err := s.stateDb.Get(key); err == nil {
		r := &Resource{}
		if err = r.DeSerialize(data); err == nil {
			return r, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (s *worldState) Delete(key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.stateDb.Delete(key)
}

func (s *worldState) Put(r *Resource) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if data, err := r.Serialize(); err == nil {
		return s.stateDb.Put(r.Key, data)
	} else {
		return err
	}
}

func (s *worldState) Reset() error {
	s.lock.Lock()
	defer s.lock.Unlock()
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
		return &worldState{
			stateDb: stateDb,
		}, nil
	} else {
		return nil, fmt.Errorf("could not instantiate DB")
	}
}
