package p2p

import (
    "testing"
    "fmt"
    "time"
)

func testConfig() Config {
	return Config{}
}

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


func TestDEVp2pRunners(t *testing.T) {
	// create an instance of DEVp2p layer
	called1st := false
	count := 0
	layer := NewDEVp2pLayer(testConfig(), func(peer Peer) error {
			count += 1
			called1st = true
			time.Sleep(100 * time.Millisecond)
			return nil
	})
	layer.AddRunner(func(peer Peer) error {
			// yet another runner
			count += 1
			time.Sleep(100 * time.Millisecond)
			return nil
	})
	layer.AddRunner(func(peer Peer) error {
			// yet another runner
			count += 1
			time.Sleep(100 * time.Millisecond)
			return nil
	})
	layer.AddRunner(func(peer Peer) error {
			// yet another runner
			count += 1
			time.Sleep(100 * time.Millisecond)
			return nil
	})
	layer.runner(nil, nil)
	if count != len(layer.runners) {
		t.Errorf("Failed to call P2P layer's all runners, expected %d, found %d", len(layer.runners), count)
	}
	if !called1st {
		t.Errorf("Failed to set flag in the first runner")
	}
}
