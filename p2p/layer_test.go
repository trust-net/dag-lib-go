package p2p

import (
    "testing"
    "fmt"
)

func TestDEVp2pInstance(t *testing.T) {
	var p2p Layer
	// test and validate p2pImpl is a P2P
	p2p = NewDEVp2pLayer(testConfig(), func(peer Peer) error {return nil})
	if impl, ok := p2p.(*layerDEVp2p); ok {
		fmt.Printf("p2p layer type: %T\n", impl)
	} else {
		t.Errorf("Failed to cast P2P layer into DEVp2p implementation")
	}
}


func TestDEVp2pRunner(t *testing.T) {
	// create an instance of DEVp2p layer
	called := false
	layer := NewDEVp2pLayer(testConfig(), func(peer Peer) error {
			called = true
			return nil
	})
	layer.runner(nil, nil)
	if !called {
		t.Errorf("Callback did not get called")
	}
}
