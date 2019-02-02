// Copyright 2018-2019 The trust-net Authors
package p2p

import (
	"os"
	"testing"
)

func testECDSAKey() ECDSAKey {
	return ECDSAKey{
		Curve: "S256",
		D:     []byte("IA1zVGm82PfnqmNZCI7Hg6H3quk3aFxj22K4vG/cNu4="),
		X:     []byte("yJqeojgViDX0NccWrkbrpqar/CWEJIiTgbs6HV2hblY="),
		Y:     []byte("8MYmGOxogoLBjrq24ehS6gf1KBFFjNFHbv+1WgWuG3A="),
	}
}

func TestEcdsaKey(t *testing.T) {
	config := Config{
		KeyFile: "key_file.json",
		KeyType: "ECDSA_S256",
	}
	if _, err := config.key(); err != nil {
		t.Errorf("Failed to get ECDSA key from config: %s", err)
	}
}

func TestEcdsaKeyTypeValidations(t *testing.T) {
	config := Config{
		KeyFile: "key_file.json",
		KeyType: "random",
	}
	if _, err := config.key(); err == nil {
		t.Errorf("did not validate KeyType, should be ECDSA_S256")
	}
}

func TestEcdsaKeyFileRequiredValidations(t *testing.T) {
	config := Config{
		KeyType: "ECDSA_S256",
	}
	if _, err := config.key(); err == nil {
		t.Errorf("did not validate KeyFile, required parameter")
	}
}

func TestEcdsaKeyFileAccessValidations(t *testing.T) {
	config := Config{
		KeyFile: "path/to/non/accessible.json",
		KeyType: "ECDSA_S256",
	}
	if _, err := config.key(); err == nil {
		t.Errorf("did not validate access to key file")
	}
}

func TestEcdsaKeyParseValidations(t *testing.T) {
	config := Config{
		KeyFile: "invalid_key_file.json",
		KeyType: "ECDSA_S256",
	}
	if _, err := config.key(); err == nil {
		t.Errorf("did not handle key parse error")
	}
}

func TestEcdsaKeyNonExisting(t *testing.T) {
	keyFile := "non_existing_key_file.json"
	os.Remove(keyFile)
	defer os.Remove(keyFile)
	config := Config{
		KeyFile: keyFile,
		KeyType: "ECDSA_S256",
	}
	if _, err := config.key(); err != nil {
		t.Errorf("Failed to get ECDSA key from config: %s", err)
	}
}

func TestNatEnabled(t *testing.T) {
	config := Config{
		NAT: true,
	}
	if nat := config.nat(); nat == nil {
		t.Errorf("Failed to enable NAT from config")
	}
}

func TestNatDisabled(t *testing.T) {
	config := Config{
		NAT: false,
	}
	if nat := config.nat(); nat != nil {
		t.Errorf("Did not disable NAT from config")
	}
}

func TestNatDefault(t *testing.T) {
	config := Config{}
	if nat := config.nat(); nat != nil {
		t.Errorf("By default did not disable NAT from config")
	}
}

func TestBootnodes(t *testing.T) {
	config := Config{
		Bootnodes: []string{"enode://210cc150e40c5f9ea68d6e9c97d5fd01bc45c71c4aa41f3126d39b80d36e368b8bf51f2b27ce5f2dbac7f36d862517c57ac0f3bd853b3300910fee17546f39ba@192.168.1.114:57743"},
	}
	if bootnodes := config.bootnodes(); bootnodes == nil {
		t.Errorf("Failed to read bootnodes from config")
	}
}

func TestNoBootnodes(t *testing.T) {
	config := Config{}
	if bootnodes := config.bootnodes(); bootnodes != nil {
		t.Errorf("Unexpected bootnodes from config")
	}
}

func TestInvalidBootnodes(t *testing.T) {
	config := Config{
		Bootnodes: []string{
			"enode://210cc150e40c5f9ea68d6e9c97d5fd01bc45c71c4aa41f3126d39b80d36e368b8bf51f2b27ce5f2dbac7f36d862517c57ac0f3bd853b3300910fee17546f39ba@192.168.1.114:57743",
			"enode://invalid_node",
		},
	}
	if bootnodes := config.bootnodes(); len(bootnodes) != 1 {
		t.Errorf("Failed to handle invalid bootnode from config")
	}
}

func TestToDEVp2pConfigInvalidKey(t *testing.T) {
	config := TestConfig()
	config.KeyFile = "invalid_key_file.json"
	if _, err := config.toDEVp2pConfig(); err == nil {
		t.Errorf("Expected toDEVp2pConfig to fail due to invalid key")
	}
}

func TestToDEVp2pConfigNoMaxPeers(t *testing.T) {
	config := TestConfig()
	config.MaxPeers = 0
	if _, err := config.toDEVp2pConfig(); err == nil {
		t.Errorf("Expected toDEVp2pConfig to fail due to missing Max Peers")
	}
}

func TestToDEVp2pConfigNoProtocolName(t *testing.T) {
	config := TestConfig()
	config.ProtocolName = ""
	if _, err := config.toDEVp2pConfig(); err == nil {
		t.Errorf("Expected toDEVp2pConfig to fail due to missing Max Peers")
	}
}

func TestToDEVp2pConfigNoName(t *testing.T) {
	config := TestConfig()
	config.Name = ""
	if _, err := config.toDEVp2pConfig(); err == nil {
		t.Errorf("Expected toDEVp2pConfig to fail due to missing Name")
	}
}

func TestListenAddrNoPort(t *testing.T) {
	config := TestConfig()
	config.ListenAddr = "test"
	config.Port = ""
	if addr := config.listenAddr(); addr != "test" {
		t.Errorf("Incorrect listen address, expected: %s, got: %s", "test", addr)
	}
}

func TestListenAddrNoListenAddr(t *testing.T) {
	config := TestConfig()
	config.ListenAddr = ""
	config.Port = "1234"
	if addr := config.listenAddr(); addr != ":1234" {
		t.Errorf("Incorrect listen address, expected: %s, got: %s", ":1234", addr)
	}
}
