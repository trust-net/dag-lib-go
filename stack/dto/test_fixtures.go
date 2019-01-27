package dto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trust-net/dag-lib-go/common"
	"math/big"
	mrand "math/rand"
)

func TestTransaction() *transaction {
	return &transaction{
		Payload:   []byte("test data"),
		Signature: []byte("test signature"),
		TxAnchor: &Anchor{
			NodeId:    []byte("test node ID"),
			ShardId:   []byte("test shard"),
			Submitter: []byte("test submitter"),
		},
	}
}

func TestAnchor() *Anchor {
	return &Anchor{
		NodeId:          []byte("test node ID"),
		ShardId:         []byte("test shard"),
		Submitter:       []byte("test submitter"),
		ShardSeq:        0x01,
		Weight:          0x01,
		SubmitterSeq:    0x01,
		SubmitterLastTx: [64]byte{},
	}
}

type Submitter struct {
	Key    *ecdsa.PrivateKey
	Id     []byte
	Seq    uint64
	LastTx [64]byte
}

func (s *Submitter) NewTransaction(txAnchor *Anchor, data string) Transaction {
	tx := &transaction{
		Payload:  []byte(data),
		TxAnchor: txAnchor,
	}
	// sign the test payload using SHA256 digest and ECDSA private key
	type signature struct {
		R *big.Int
		S *big.Int
	}
	sig := signature{}
	// use submitter's sequence as nonce in the payload signature
	hash := sha256.Sum256(append(common.Uint64ToBytes(txAnchor.SubmitterSeq), tx.Payload...))
	sig.R, sig.S, _ = ecdsa.Sign(rand.Reader, s.Key, hash[:])
	tx.Signature = append(sig.R.Bytes(), sig.S.Bytes()...)
	return tx
}

func TestSubmitter() *Submitter {
	// create a new ECDSA key for submitter client
	key, _ := crypto.GenerateKey()
	id := crypto.FromECDSAPub(&key.PublicKey)
	return &Submitter{
		Key:    key,
		Id:     id,
		Seq:    1,
		LastTx: [64]byte{},
	}

}

func TestSignedTransaction(data string) *transaction {
	tx := &transaction{
		Payload:  []byte(data),
		TxAnchor: TestAnchor(),
	}
	// create a new ECDSA key for submitter client
	key, _ := crypto.GenerateKey()
	tx.TxAnchor.Submitter = crypto.FromECDSAPub(&key.PublicKey)

	// sign the test payload using SHA512 hash and ECDSA private key
	type signature struct {
		R *big.Int
		S *big.Int
	}
	s := signature{}
	hash := sha256.Sum256(tx.Payload)
	s.R, s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	tx.Signature = append(s.R.Bytes(), s.S.Bytes()...)
	return tx
}

func RandomHash() [64]byte {
	hash := [64]byte{}
	for i := 0; i < 64; i++ {
		hash[i] = byte(mrand.Int31n(255))
	}
	return hash
}
