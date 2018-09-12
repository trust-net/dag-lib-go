package p2p

import (
    "testing"
    "os"
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
		KeyFile: "key_file.json",
		KeyType: "ECDSA_S256",
	}
	if key := config.key(); key == nil {
		t.Errorf("Failed to get ECDSA key from config")
	}
}

func TestEcdsaKeyTypeValidations(t *testing.T) {
	config := Config {
		KeyFile: "key_file.json",
		KeyType: "random",
	}
	if config.key() != nil {
		t.Errorf("did not validate KeyType, should be ECDSA_S256")
	}
}

func TestEcdsaKeyFileRequiredValidations(t *testing.T) {
	config := Config {
		KeyType: "ECDSA_S256",
	}
	if config.key() != nil {
		t.Errorf("did not validate KeyFile, required parameter")
	}
}

func TestEcdsaKeyFileAccessValidations(t *testing.T) {
	config := Config {
		KeyFile: "path/to/non/accessible.json",
		KeyType: "ECDSA_S256",
	}
	if config.key() != nil {
		t.Errorf("did not validate access to key file")
	}
}

func TestEcdsaKeyParseValidations(t *testing.T) {
	config := Config {
		KeyFile: "invalid_key_file.json",
		KeyType: "ECDSA_S256",
	}
	if config.key() != nil {
		t.Errorf("did not handle key parse error")
	}
}

func TestEcdsaKeyNonExisting(t *testing.T) {
	keyFile := "non_existing_key_file.json"
	os.Remove(keyFile)
	defer os.Remove(keyFile)
	config := Config {
		KeyFile: keyFile,
		KeyType: "ECDSA_S256",
	}
	if key := config.key(); key == nil {
		t.Errorf("Failed to get ECDSA key from config")
	}
}
