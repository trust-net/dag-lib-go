package p2p

import (
    "testing"
    "fmt"
    "crypto/ecdsa"
    "crypto/sha512"
    "crypto/rand"
    "github.com/trust-net/go-trust-net/common"
    "github.com/ethereum/go-ethereum/crypto"
)

func TestDEVp2pInstance(t *testing.T) {
	var p2p Layer
	var err error
	// test and validate p2pImpl is a P2P
	conf := TestConfig()
	p2p, err = NewDEVp2pLayer(conf, func(peer Peer) error {return nil})
	if err != nil {
		t.Errorf("Failed to get P2P layer instance: %s", err)
	}
	if impl, ok := p2p.(*layerDEVp2p); ok {
		fmt.Printf("p2p layer type: %T\n", impl)
	} else {
		t.Errorf("Failed to cast P2P layer into DEVp2p implementation")
	}
	// p2p node's ID should be initialized correctly
	if string(p2p.Id()) != string(crypto.FromECDSAPub(&p2p.(*layerDEVp2p).conf.PrivateKey.PublicKey)) {
		t.Errorf("Did not initialize p2p node's ID")
	}
}

func TestDEVp2pInstanceBadConfig(t *testing.T) {
	_, err := NewDEVp2pLayer(Config{}, func(peer Peer) error {return nil})
	if err == nil {
		t.Errorf("Expected no instance due to bad config")
	}
}

func TestDEVp2pRunner(t *testing.T) {
	// create an instance of DEVp2p layer
	called := false
	layer,_ := NewDEVp2pLayer(TestConfig(), func(peer Peer) error {
			called = true
			return nil
	})
	layer.runner(nil, nil)
	if !called {
		t.Errorf("Callback did not get called")
	}
}

func TestDEVp2pSign(t *testing.T) {
	// create an instance of the p2p layer
	conf := TestConfig()
	p2p, _ := NewDEVp2pLayer(conf, func(peer Peer) error {return nil})

	// create a test payload
	payload := []byte("test data")

	// get key from test config
	key, _ := conf.key()

	// sign the test payload
	if p2pSignature, err := p2p.Sign(payload); err != nil {
		t.Errorf("Failed to get p2p signature: %s", err)
	} else {
		// regenerate signature parameters
		s := signature{}
		if err := common.Deserialize(p2pSignature, &s); err != nil {
			t.Errorf("Failed to parse signature: %s", err)
			return
		}
		// we want to validate the hash of the payload
		hash := sha512.Sum512(payload)
		// validate signature of payload
		if !ecdsa.Verify(&key.PublicKey, hash[:], s.R, s.S) {
			t.Errorf("signature validation failed")
		}
	}
}

func TestDEVp2pVerify(t *testing.T) {
	// create an instance of the p2p layer
	p2p, _ := NewDEVp2pLayer(TestConfig(), func(peer Peer) error {return nil})

	// create a test payload
	payload := []byte("test data")

	// create a new ECDSA key
	key, _ := crypto.GenerateKey()
	id := crypto.FromECDSAPub(&key.PublicKey)

	// sign the test payload using SHA512 hash and ECDSA private key
	s := signature{}
	hash := sha512.Sum512(payload)
	s.R,s.S, _ = ecdsa.Sign(rand.Reader, key, hash[:])
	sign, _ := common.Serialize(s)

	// validate that p2p layer can verify the signature
	if !p2p.Verify(payload, sign, id) {
		t.Errorf("Failed to verify signature")
	}
}
