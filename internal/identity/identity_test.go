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

func TestRotateGeneratesNewKeyPair(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", dir+"/agent_identity.key")

	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	newKP, pending, err := Rotate()
	if err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}
	defer pending.Rollback() //nolint:errcheck

	if original.PublicKey.Equal(newKP.PublicKey) {
		t.Error("Rotate() returned the same public key as the original")
	}
}

func TestCommitReplacesOldKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", dir+"/agent_identity.key")

	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	newKP, pending, err := Rotate()
	if err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	if err := pending.Commit(); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Commit() error: %v", err)
	}

	if !loaded.PublicKey.Equal(newKP.PublicKey) {
		t.Error("Load() after Commit() returned old key, expected new key")
	}
	if loaded.PublicKey.Equal(original.PublicKey) {
		t.Error("Load() after Commit() returned original key, expected new key")
	}
}

func TestRollbackDeletesStagedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", dir+"/agent_identity.key")

	if _, err := Generate(); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	_, pending, err := Rotate()
	if err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	tmpPath := pending.tmpPath
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatalf("staged file should exist before Rollback, got: %v", err)
	}

	if err := pending.Rollback(); err != nil {
		t.Fatalf("Rollback() error: %v", err)
	}

	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("staged file should not exist after Rollback")
	}
}

func TestRotateIsIdempotentOnFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", dir+"/agent_identity.key")

	original, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	_, pending, err := Rotate()
	if err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	// Simulate server error — rollback
	if err := pending.Rollback(); err != nil {
		t.Fatalf("Rollback() error: %v", err)
	}

	// Load should still return the original key
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Rollback() error: %v", err)
	}
	if !loaded.PublicKey.Equal(original.PublicKey) {
		t.Error("Load() after Rollback() returned wrong key; original should be unchanged")
	}
}
