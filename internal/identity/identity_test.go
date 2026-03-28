package identity

import (
	"crypto/ed25519"
	"os"
	"testing"
)

func TestGenerateCreatesKeyPair(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/agent_identity.key")

	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(kp.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey size: want %d, got %d", ed25519.PublicKeySize, len(kp.PublicKey))
	}
	if len(kp.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("PrivateKey size: want %d, got %d", ed25519.PrivateKeySize, len(kp.PrivateKey))
	}
	// Verify signing round-trip
	msg := []byte("test-message")
	sig := ed25519.Sign(kp.PrivateKey, msg)
	if !ed25519.Verify(kp.PublicKey, msg, sig) {
		t.Error("generated key pair failed sign/verify round-trip")
	}
}

func TestLoadRoundtrip(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/agent_identity.key")

	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !original.PublicKey.Equal(loaded.PublicKey) {
		t.Error("loaded public key does not match generated key")
	}
	if string(original.PrivateKey) != string(loaded.PrivateKey) {
		t.Error("loaded private key does not match generated key")
	}
}

func TestLoadOrGenerateIdempotent(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/agent_identity.key")

	first, err := LoadOrGenerate()
	if err != nil {
		t.Fatalf("first LoadOrGenerate() error: %v", err)
	}

	second, err := LoadOrGenerate()
	if err != nil {
		t.Fatalf("second LoadOrGenerate() error: %v", err)
	}

	if !first.PublicKey.Equal(second.PublicKey) {
		t.Error("LoadOrGenerate returned different public keys on second call")
	}
	if string(first.PrivateKey) != string(second.PrivateKey) {
		t.Error("LoadOrGenerate returned different private keys on second call")
	}
}

func TestKeyPathEnvOverride(t *testing.T) {
	customPath := t.TempDir() + "/custom_identity.key"
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", customPath)

	if KeyPath() != customPath {
		t.Errorf("KeyPath() = %q, want %q", KeyPath(), customPath)
	}

	if _, err := Generate(); err != nil {
		t.Fatalf("Generate() to custom path error: %v", err)
	}

	if _, err := os.Stat(customPath); err != nil {
		t.Errorf("key file not created at custom path %q: %v", customPath, err)
	}
}
