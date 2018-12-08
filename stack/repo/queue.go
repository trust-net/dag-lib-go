package repo

import (
	"sync"
	"errors"
)

type Queue interface {
	Push(item interface{}) error
	Count() uint64
	Pop() (interface{}, error)
	At(pos uint64) (interface{}, error)
}

type circularQ struct {
	circle []interface{}
	front  uint64
	back   uint64
	size   uint64
	count  uint64
	lock   sync.RWMutex
}

func NewQueue(size uint64) (*circularQ, error) {
	q := circularQ{
		size:   size+1,
		back:   size-1,
		front:  size,
		count:  0,
		circle: make([]interface{}, size+2, size+2),
	}
	return &q, nil
}

func (q *circularQ) Push(item interface{}) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	// rotate back to left by 1
	back := q.back
	if q.back -= 1; q.back > q.size {
		// rollover back
		q.back = q.size
	}
	// check if we are full capacity
	if q.back == q.front {
		// revert back and return error
		q.back = back
		return errors.New("queue capacity full")
	}
	// add item to back
	q.circle[back] = item
	// increment count
	q.count += 1
	return nil
}

func (q *circularQ) Count() uint64 {
	return q.count
}
func (q *circularQ) Pop() (interface{}, error) {
	q.lock.Lock()
	defer q.lock.Unlock()
	// rotate front to left by 1
	front := q.front
	if q.front -= 1; q.front > q.size {
		// rollover back
		q.front = q.size
	}
	// check if we are empty
	if q.back == q.front {
		// revert back and return error
		q.front = front
		return nil, errors.New("queue empty")
	}
	// decrement count
	q.count -= 1
	// return back the item from front
	return q.circle[q.front], nil
}

func (q *circularQ) At(pos uint64) (interface{}, error) {
	q.lock.Lock()
	defer q.lock.Unlock()
	return nil, nil
}
