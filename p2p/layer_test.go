package p2p

import (
    "testing"
    "fmt"
    "github.com/ethereum/go-ethereum/p2p"
)

func testConfig() p2p.Config {
	return p2p.Config{}
}

func TestDEVp2pInstance(t *testing.T) {
	var inst P2P
	inst = NewDEVp2pLayer(testConfig())
	if impl, ok := inst.(*p2pImpl); ok {
		fmt.Printf("p2p layer type: %T\n", impl)
	} else {
		t.Errorf("Failed to cast P2P layer into DEVp2p implementation")
	}
}

