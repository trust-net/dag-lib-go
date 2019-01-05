package common

import (
    "testing"
    "time"
)

type testError struct {}

func (e *testError) Error() string {
	return "test error"
}


func TestRunTimeBoundOnTimeout(t *testing.T) {
	// invoke a method that takes longer than timeout delay
	timeout, wait := time.Duration(1), time.Duration(5)
	if err := RunTimeBound(timeout, func() error {time.Sleep(wait*time.Second); return nil}, &testError{}); err == nil {
		t.Errorf("timed out unexpectedly")
	}
}


func TestRunTimeBoundWithinTime(t *testing.T) {
	// invoke a method that takes shorter than timeout delay
	timeout, wait := time.Duration(5), time.Duration(1)
	if err := RunTimeBound(timeout, func() error {time.Sleep(wait*time.Second); return nil}, &testError{}); err != nil {
		t.Errorf("did not timeout")
	}
}

type TestEntity struct {
	Field1 string
	Field2 int64
}

func TestSerialize(t *testing.T) {
	entity := TestEntity {"test string", 0x0045}
	if _, err := Serialize(entity); err != nil {
		t.Errorf("failed to serialize entity: %s", err)
	}
}

func TestDeseriealize(t *testing.T) {
	data, _ := Serialize(&TestEntity {"test string", 0x0045})
	var entity TestEntity
	if err := Deserialize(data, &entity); err != nil {
		t.Errorf("failed to deserialize entity: %s", err)
	} else if entity.Field1 != "test string" || entity.Field2 != 0x0045 {
		t.Errorf("Incorrect values: %s\n", entity)
	}
}

