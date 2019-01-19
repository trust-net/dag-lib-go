package common

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"time"
)

func RunTimeBound(sec time.Duration, method func() error, timeoutError error) error {
	var err error
	// create a channel to signal done
	done := make(chan struct{})
	// start a timer
	wait := time.NewTimer(sec * time.Second)
	defer wait.Stop()
	// invoke the method in a go routine
	go func() {
		err = method()
		done <- struct{}{}
	}()
	// wait for either done, or timeout
	select {
	case <-done:
		break
	case <-wait.C:
		err = timeoutError
	}
	return err
}

func RunTimeBoundSec(sec int, method func() error, timeoutError error) error {
	return RunTimeBound(time.Duration(sec), method, timeoutError)
}

func Uint64ToBytes(value uint64) []byte {
	var byte8 [8]byte
	binary.BigEndian.PutUint64(byte8[:], value)
	return byte8[:]
}

func BytesToUint64(value []byte) uint64 {
	byte8 := make([]byte, 8, 8)
	copy(byte8, value)
	return binary.BigEndian.Uint64(byte8)
}

func Serialize(entity interface{}) ([]byte, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	if err := e.Encode(entity); err != nil {
		return []byte{}, err
	} else {
		return b.Bytes(), nil
	}
}

func Deserialize(data []byte, entity interface{}) error {
	b := bytes.Buffer{}
	b.Write(data)
	d := gob.NewDecoder(&b)
	return d.Decode(entity)
}
