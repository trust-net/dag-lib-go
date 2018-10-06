package stack

import (
	"github.com/trust-net/dag-lib-go/stack/p2p"
)


func TestTransaction() *Transaction {
	return &Transaction {
		Payload: []byte("test data"),
		Signature: []byte("test signature"),
		AppId: []byte("test app ID"),
		Submitter: []byte("test submitter"),
	}
}

func TestAppConfig() AppConfig {
	return AppConfig {
		AppId: []byte("test app ID"),
		ShardId: []byte ("test shard"),
		Name: "test app",
		Version: 1234,
	}
}

func testP2PConfig() p2p.Config {
	return p2p.TestConfig()
}

func testP2PLayer(name string) *p2p.MockP2P {
	return p2p.TestP2PLayer(name)
}
