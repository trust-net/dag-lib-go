package stack

import (
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
