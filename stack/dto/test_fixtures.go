package dto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trust-net/go-trust-net/common"
	"math/big"
)


func TestTransaction() *Transaction {
	return &Transaction{
		Payload:   []byte("test data"),
		Signature: []byte("test signature"),
		AppId:     []byte("test app ID"),
		ShardId:   []byte("test shard"),
		Submitter: []byte("test submitter"),
	}
}

func TestSignedTransaction(data string) *Transaction {
	tx := &Transaction{
		Payload: []byte(data),
		ShardId: []byte("test shard"),
		AppId:   []byte("test app ID"),
	}

	// create a new ECDSA key for submitter client
	key, _ := crypto.GenerateKey()
	tx.Submitter = crypto.FromECDSAPub(&key.PublicKey)

	// sign the test payload using SHA512 hash and ECDSA private key
	type signature struct {
		R *big.Int
		S *big.Int
	}
	s := signature{}
	hash := sha512.Sum512(tx.Payload)
	s.R, s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	tx.Signature, _ = common.Serialize(s)
	return tx
}
