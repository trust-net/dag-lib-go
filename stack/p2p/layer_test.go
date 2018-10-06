package p2p

import (
    "testing"
    "fmt"
)

func TestDEVp2pInstance(t *testing.T) {
	var p2p Layer
	var err error
	// test and validate p2pImpl is a P2P
	p2p, err = NewDEVp2pLayer(TestConfig(), func(peer Peer) error {return nil})
	if err != nil {
		t.Errorf("Failed to get P2P layer instance: %s", err)
	}
	if impl, ok := p2p.(*layerDEVp2p); ok {
		fmt.Printf("p2p layer type: %T\n", impl)
	} else {
		t.Errorf("Failed to cast P2P layer into DEVp2p implementation")
	}
}

func TestDEVp2pInstanceBadConfig(t *testing.T) {
	_, err := NewDEVp2pLayer(Config{}, func(peer Peer) error {return nil})
	if err == nil {
		t.Errorf("Expected no instance due to bad config")
	}
}


func TestDEVp2pRunner(t *testing.T) {
	// create an instance of DEVp2p layer
	called := false
	layer,_ := NewDEVp2pLayer(TestConfig(), func(peer Peer) error {
			called = true
			return nil
	})
	layer.runner(nil, nil)
	if !called {
		t.Errorf("Callback did not get called")
	}
}
