// Copyright 2018-2019 The trust-net Authors
package repo

import (
	"testing"
)

func TestQueueInitiatization(t *testing.T) {
	var q Queue
	var cq *circularQ
	var err error

	if q, err = NewQueue(20); err != nil {
		t.Errorf("Failed to instantiate: %s", err)
	} else {
		cq = q.(*circularQ)
	}

	if cq.size != 21 {
		t.Errorf("Incorrect initial size: %d", cq.size)
	}

	if cq.count != 0 {
		t.Errorf("Incorrect initial count: %d", cq.count)
	}

	if cq.front != 20 {
		t.Errorf("Incorrect initial front: %d", cq.front)
	}

	if cq.back != 19 {
		t.Errorf("Incorrect initial back: %d", cq.back)
	}

	if len(cq.circle) != 20+2 {
		t.Errorf("Incorrect initial circle size: %d", len(cq.circle))
	}
}

func TestPushPopOrder(t *testing.T) {
	q, _ := NewQueue(20)
	q.Push("test data 1")
	q.Push("test data 2")
	item, _ := q.Pop()
	if item != "test data 1" {
		t.Errorf("Incorrect popped item: %s", item)
	}
	item, _ = q.Pop()
	if item != "test data 2" {
		t.Errorf("Incorrect popped item: %s", item)
	}
}

func TestFirstPush(t *testing.T) {
	q, _ := NewQueue(20)
	if err := q.Push("data"); err != nil {
		t.Errorf("Failed to push: %s", err)
	}

	// front stays the same
	if q.front != 20 {
		t.Errorf("Incorrect first push front: %d", q.front)
	}

	// back should move left
	if q.back != 18 {
		t.Errorf("Incorrect first push back: %d", q.back)
	}

	if q.count != 1 {
		t.Errorf("Incorrect first push count: %d", q.count)
	}
}

func TestRolloverPush(t *testing.T) {
	q, _ := NewQueue(20)
	// change back to left most position, 0
	q.back = 0
	// change front to make some space in list
	q.front = 15
	if err := q.Push("data"); err != nil {
		t.Errorf("Failed to push: %s", err)
	}

	// front stays the same
	if q.front != 15 {
		t.Errorf("Incorrect rollover push front: %d", q.front)
	}

	// back should roll over to right most position
	if q.back != 21 {
		t.Errorf("Incorrect rollover push back: %d", q.back)
	}
}

func TestFullPushRollover(t *testing.T) {
	q, _ := NewQueue(20)
	// change back to left most position, 0
	q.back = 0
	// keep front to right most position
	q.front = 21
	if err := q.Push("data"); err == nil {
		t.Errorf("Did not fail on full capacity push")
	}

	// front stays the same
	if q.front != 21 {
		t.Errorf("Incorrect full capacity push front: %d", q.front)
	}

	// back should stay the same
	if q.back != 0 {
		t.Errorf("Incorrect full capacity push back: %d", q.back)
	}
}

func TestFullPushNormal(t *testing.T) {
	q, _ := NewQueue(20)
	// change front to somewhere in middle
	q.front = 10
	// change back to immediately right of front
	q.back = 11
	if err := q.Push("data"); err == nil {
		t.Errorf("Did not fail on full capacity push")
	}

	// front stays the same
	if q.front != 10 {
		t.Errorf("Incorrect full capacity push front: %d", q.front)
	}

	// back should stay the same
	if q.back != 11 {
		t.Errorf("Incorrect full capacity push back: %d", q.back)
	}
}

func TestEmptyPop(t *testing.T) {
	q, _ := NewQueue(20)
	if _, err := q.Pop(); err == nil {
		t.Errorf("Did not fail on empty pop")
	}

	// front stays the same
	if q.front != 20 {
		t.Errorf("Incorrect first pop front: %d", q.front)
	}

	// back should stay same
	if q.back != 19 {
		t.Errorf("Incorrect first pop back: %d", q.back)
	}

	if q.count != 0 {
		t.Errorf("Incorrect first pop count: %d", q.count)
	}
}

func TestRolloverPop(t *testing.T) {
	q, _ := NewQueue(20)
	// change back to make some space in list
	q.back = 15
	// set count to some value
	q.count = 5
	// change front to left most position, 1
	q.front = 0
	if _, err := q.Pop(); err != nil {
		t.Errorf("Failed to pop: %s", err)
	}

	// front should roll over to right most position
	if q.front != 21 {
		t.Errorf("Incorrect rollover pop front: %d", q.front)
	}

	// count should go down by 1
	if q.count != 4 {
		t.Errorf("Incorrect pop count: %d", q.count)
	}

	// back stays the same
	if q.back != 15 {
		t.Errorf("Incorrect rollover pop back: %d", q.back)
	}
}

func TestEmptyPopRollover(t *testing.T) {
	q, _ := NewQueue(20)
	// change front to left most position, 0
	q.front = 0
	// keep back to right most position
	q.back = 21
	if _, err := q.Pop(); err == nil {
		t.Errorf("Did not fail on empty pop")
	}

	// front stays the same
	if q.front != 0 {
		t.Errorf("Incorrect empty pop front: %d", q.front)
	}

	// back should stay the same
	if q.back != 21 {
		t.Errorf("Incorrect empty pop back: %d", q.back)
	}
}

func TestEmptyPopNormal(t *testing.T) {
	q, _ := NewQueue(20)
	// change back to somewhere in middle
	q.back = 10
	// change front to immediately right of back
	q.front = 11
	if _, err := q.Pop(); err == nil {
		t.Errorf("Did not fail on empty pop")
	}

	// front stays the same
	if q.front != 11 {
		t.Errorf("Incorrect empty pop front: %d", q.front)
	}

	// back should stay the same
	if q.back != 10 {
		t.Errorf("Incorrect empty pop back: %d", q.back)
	}
}

func TestFillUpAndDrain(t *testing.T) {
	size := uint64(100)
	q, _ := NewQueue(size)

	// add some small number of items, so it will roll over in the test

	for i := 0; i < 20; i++ {
		q.Push("test data")
	}

	// now run the fill up and drain in a loop

	for i := 0; i < 5; i++ {
		// fill up queue
		for {
			if err := q.Push("test data"); err != nil {
				break
			}
		}

		if q.Count() != size {
			t.Errorf("Incorrect fill up count: %d", q.Count())
		}

		// drain the queue
		for {
			if data, err := q.Pop(); err != nil {
				break
			} else if data != "test data" {
				t.Errorf("%d: Incorrect popped value: %s", q.Count(), data)
			}
		}

		if q.Count() != 0 {
			t.Errorf("Incorrect drained up count: %d", q.Count())
		}
	}
}
