package p2p

import (
    "testing"
    "fmt"
)

func testConfig() Config {
	return Config{}
}

func TestDEVp2pInstance(t *testing.T) {
	var p2p P2P
	// test and validate p2pImpl is a P2P
	p2p = NewDEVp2pLayer(testConfig())
	if impl, ok := p2p.(*p2pImpl); ok {
		fmt.Printf("p2p layer type: %T\n", impl)
	} else {
		t.Errorf("Failed to cast P2P layer into DEVp2p implementation")
	}
}

