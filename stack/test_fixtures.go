package stack

import (
	"math/big"
    "crypto/ecdsa"
    "crypto/sha512"
    "crypto/rand"
    "github.com/trust-net/go-trust-net/common"
    "github.com/ethereum/go-ethereum/crypto"
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

func TestSignedTransaction(data string) *Transaction {
	tx := &Transaction {
		Payload: []byte(data),
	}

	// create a new ECDSA key
	key, _ := crypto.GenerateKey()
	tx.AppId = crypto.FromECDSAPub(&key.PublicKey)

	// sign the test payload using SHA512 hash and ECDSA private key
	type signature struct {
		R *big.Int
		S *big.Int
	}
	s := signature{}
	hash := sha512.Sum512(tx.Payload)
	s.R,s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	tx.Signature, _ = common.Serialize(s)
	return tx
}
