// Copyright 2018-2019 The trust-net Authors
// a basic set implementation
package common

import (
	"sync"
)

var marker = struct{}{}

type Set struct {
	data map[interface{}]struct{}
	lock sync.RWMutex
}

func NewSet() *Set {
	set := Set{
		data: make(map[interface{}]struct{}),
		lock: sync.RWMutex{},
	}
	return &set
}

// get size of the set
func (set *Set) Size() int {
	set.lock.Lock()
	defer set.lock.Unlock()
	return len(set.data)
}

// add group of elements into the set
func (set *Set) Add(items ...interface{}) {
	if len(items) == 0 {
		return
	}
	set.lock.Lock()
	defer set.lock.Unlock()
	for _, item := range items {
		set.data[item] = marker
	}
}

func (set *Set) Remove(items ...interface{}) {
	if len(items) == 0 {
		return
	}
	set.lock.Lock()
	defer set.lock.Unlock()
	for _, item := range items {
		delete(set.data, item)
	}
}

func (set *Set) Has(item interface{}) bool {
	set.lock.RLock()
	defer set.lock.RUnlock()
	_, has := set.data[item]
	return has
}

func (set *Set) Pop() interface{} {
	set.lock.Lock()
	defer set.lock.Unlock()
	for item, _ := range set.data {
		// delete item from set
		delete(set.data, item)
		// return the first item popped
		return item
	}
	return nil
}
