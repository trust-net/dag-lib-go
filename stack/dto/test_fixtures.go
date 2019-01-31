package dto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	mrand "math/rand"
)

func TestTransaction() *transaction {
	return &transaction{
		TxRequest: TestRequest(),
		TxAnchor: TestAnchor(),
	}
}

func TestRequest() *TxRequest {
	return &TxRequest{
		// payload for transaction's operations
		Payload: []byte("test data"),
		// shard id for the transaction
		ShardId: []byte("test shard"),
		// submitter's last transaction
		LastTx: [64]byte{},
		// Submitter's public ID
		SubmitterId: []byte("test submitter"),
		// submitter's transaction sequence
		SubmitterSeq: 0x01,
		// a padding to meet challenge for network's DoS protection
		Padding: 0x00,
		// signature of the transaction request's contents using submitter's private key
		Signature: []byte("test request signature"),
	}
}

func TestAnchor() *Anchor {
	return &Anchor{
		NodeId:          []byte("test node ID"),
		ShardSeq:        0x01,
		Weight:          0x01,
	}
}

type Submitter struct {
	Key    *ecdsa.PrivateKey
	Id     []byte
	Seq    uint64
	LastTx [64]byte
}

func (s *Submitter) NewTransaction(txAnchor *Anchor, data string) *transaction {
	return &transaction{
		TxRequest:  s.NewRequest(data),
		TxAnchor: txAnchor,
	}
}

func (s *Submitter) NewRequest(data string) *TxRequest {
	req := &TxRequest{
		// payload for transaction's operations
		Payload: []byte(data),
		// shard id for the transaction
		ShardId: []byte("test shard"),
		// submitter's last transaction
		LastTx: s.LastTx,
		// Submitter's public ID
		SubmitterId: s.Id,
		// submitter's transaction sequence
		SubmitterSeq: s.Seq,
		// a padding to meet challenge for network's DoS protection
		Padding: 0x00,
	}

	// sign the request using SHA256 digest and ECDSA private key
	type signature struct {
		R *big.Int
		S *big.Int
	}
	sig := signature{}
	// sign the request
	hash := sha256.Sum256(req.Bytes())
	sig.R, sig.S, _ = ecdsa.Sign(rand.Reader, s.Key, hash[:])
	req.Signature = append(sig.R.Bytes(), sig.S.Bytes()...)
	return req
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
	return TestSubmitter().NewTransaction(TestAnchor(), data)
}

func RandomHash() [64]byte {
	hash := [64]byte{}
	for i := 0; i < 64; i++ {
		hash[i] = byte(mrand.Int31n(255))
	}
	return hash
}
