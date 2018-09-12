package p2p

import (
    "testing"
    "fmt"
)

func testECDSAKey() ECDSAKey {
	return ECDSAKey{
			Curve: "S256",
			D: []byte("IA1zVGm82PfnqmNZCI7Hg6H3quk3aFxj22K4vG/cNu4="),
			X: []byte("yJqeojgViDX0NccWrkbrpqar/CWEJIiTgbs6HV2hblY="),
			Y: []byte("8MYmGOxogoLBjrq24ehS6gf1KBFFjNFHbv+1WgWuG3A="),
	}
}

func TestEcdsaKey(t *testing.T) {
	config := Config {
		PrivateKey: testECDSAKey(),
	}
	if key := config.key(); key == nil {
		t.Errorf("Failed to get ECDSA key from config")
	} else {
		fmt.Printf("Public Key: %s\n", *key)
	}
}

func TestEcdsaKeyCurveValidations(t *testing.T) {
	config := Config {
		PrivateKey: testECDSAKey(),
	}
	config.PrivateKey.Curve = "random"
	if config.key() != nil {
		t.Errorf("did not validate curve = S256")
	}
}
