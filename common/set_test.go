package common

import (
    "testing"
)

func TestZeroAdd(t *testing.T) {
	uut := NewSet()
	uut.Add()
	if uut.Size() != 0 {
		t.Errorf("Expected: %d, Actual: %d", 0, uut.Size())
	}
}

func TestDuplicateAdd(t *testing.T) {
	uut := NewSet()
	uut.Add("1", 2, '3', 2, "1")
	if uut.Size() != 3 {
		t.Errorf("Expected: %d, Actual: %d", 2, uut.Size())
	}
}

func TestMultiAdd(t *testing.T) {
	uut := NewSet()
	uut.Add("1", 2, struct{}{}, 'c')
	if uut.Size() != 4 {
		t.Errorf("Expected: %d, Actual: %d", 4, uut.Size())
	}
}

func TestHasAdd(t *testing.T) {
	uut := NewSet()
	items := []interface{}{"1", 2, struct{}{}, 'c'}
	uut.Add(items...)
	for _, item := range items {
		if !uut.Has(item) {
			t.Errorf("Expected: %s, Not found", item)
		}
	}
}

func TestPop(t *testing.T) {
	uut := NewSet()
	items := []interface{}{1, "2"}
	uut.Add(items...)
	popped := uut.Pop()
	// item popped is not predictable, so cannot force check
	if popped != 1 && popped != "2" {
		t.Errorf("Expected to pop: %s, Found: %s", "1 or 2", popped)
	}
	if uut.Size() != 1 {
		t.Errorf("Expected Size: %d, Actual: %d", 1, uut.Size())
	}
}

func TestMultiRemove(t *testing.T) {
	uut := NewSet()
	uut.Add("1", 2, struct{}{}, 'c')
	uut.Remove("1", 'c')
	if uut.Size() != 2 {
		t.Errorf("Expected: %d, Actual: %d", 2, uut.Size())
	}
}

func TestNonExistingRemove(t *testing.T) {
	uut := NewSet()
	uut.Add("1", 2, struct{}{}, 'c')
	uut.Remove(1, "2", 'c')
	if uut.Size() != 3 {
		t.Errorf("Expected: %d, Actual: %d", 3, uut.Size())
	}
}