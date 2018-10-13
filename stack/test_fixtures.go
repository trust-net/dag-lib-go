package stack

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha512"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trust-net/dag-lib-go/stack/shard"
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

func TestAppConfig() AppConfig {
	return AppConfig{
		AppId:   []byte("test app ID"),
		ShardId: []byte("test shard"),
		Name:    "test app",
	}
}

func TestSignedTransaction(data string) *Transaction {
	tx := &Transaction{
		Payload: []byte(data),
		ShardId: []byte("test shard"),
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
	s.R, s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	tx.Signature, _ = common.Serialize(s)
	return tx
}

type mockSharder struct {
	IsRegistered    bool
	ShardId         []byte
	TxHandlerCalled bool
	TxHandler       func(tx *shard.Transaction) error
}

func (s *mockSharder) Register(shardId []byte, txHandler func(tx *shard.Transaction) error) error {
	s.IsRegistered = true
	s.ShardId = shardId
	s.TxHandler = txHandler
	return nil
}

func (s *mockSharder) Unregister() error {
	s.IsRegistered = false
	return nil
}

func (s *mockSharder) Handle(tx *shard.Transaction) error {
	s.TxHandlerCalled = true
	if s.TxHandler != nil {
		return s.TxHandler(tx)
	} else {
		return nil
	}
}

func NewMockSharder() *mockSharder {
	return &mockSharder{}
}
