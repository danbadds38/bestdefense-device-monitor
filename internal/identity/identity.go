package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
)

const pemType = "BESTDEFENSE AGENT PRIVATE KEY"

// ErrNotFound is returned by Load when the key file does not exist.
var ErrNotFound = errors.New("identity key file not found")

// KeyPair holds the agent's Ed25519 identity.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// PublicKeyBase64 returns the base64-encoded public key.
func (kp *KeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey)
}

// KeyPath returns the path to the private key file.
// Default: {dataDir}/agent_identity.key
// Override: BESTDEFENSE_IDENTITY_PATH env var (for tests).
func KeyPath() string {
	if p := os.Getenv("BESTDEFENSE_IDENTITY_PATH"); p != "" {
		return p
	}
	return filepath.Join(config.DataDir(), "agent_identity.key")
}

// Load reads the key pair from disk. Returns ErrNotFound if the file does not exist.
func Load() (*KeyPair, error) {
	data, err := os.ReadFile(KeyPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("reading identity key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != pemType {
		return nil, fmt.Errorf("invalid PEM block in identity key file")
	}
	if len(block.Bytes) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid key seed length: expected %d bytes, got %d", ed25519.SeedSize, len(block.Bytes))
	}

	privKey := ed25519.NewKeyFromSeed(block.Bytes)
	return &KeyPair{
		PublicKey:  privKey.Public().(ed25519.PublicKey),
		PrivateKey: privKey,
	}, nil
}

// Generate creates a new Ed25519 key pair, saves it to disk (mode 0600), and returns it.
func Generate() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 key pair: %w", err)
	}

	kp := &KeyPair{PublicKey: pub, PrivateKey: priv}
	if err := saveKeyPair(kp); err != nil {
		return nil, err
	}
	return kp, nil
}

// LoadOrGenerate loads the existing key pair or generates a new one if not present.
func LoadOrGenerate() (*KeyPair, error) {
	kp, err := Load()
	if err == nil {
		return kp, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return Generate()
}

func saveKeyPair(kp *KeyPair) error {
	seed := kp.PrivateKey.Seed()
	block := &pem.Block{
		Type:  pemType,
		Bytes: seed,
	}

	path := KeyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating identity key directory: %w", err)
	}

	return writeKeyFile(path, pem.EncodeToMemory(block))
}

// Sign produces an Ed25519 signature over the canonical request message.
// Message format: "METHOD\n/path\nhex(sha256(body))\ntimestamp_seconds"
// For GET requests with no body, pass nil or empty slice.
func (kp *KeyPair) Sign(method, path string, body []byte, timestamp int64) []byte {
	h := sha256.Sum256(body)
	msg := fmt.Sprintf("%s\n%s\n%s\n%d", method, path, hex.EncodeToString(h[:]), timestamp)
	return ed25519.Sign(kp.PrivateKey, []byte(msg))
}

// PendingRotation represents a new key pair that has been generated and staged
// at a temporary path, awaiting server confirmation before being committed.
type PendingRotation struct {
	tmpPath   string
	finalPath string
}

// Rotate generates a new Ed25519 key pair and stages it at {keyPath}.tmp.
// The new key is NOT written to the final path until Commit() is called,
// so the old key remains usable if the server rejects the rotation.
// Call Rollback() to discard the staged file on failure.
func Rotate() (*KeyPair, *PendingRotation, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating new Ed25519 key pair: %w", err)
	}

	newKP := &KeyPair{PublicKey: pub, PrivateKey: priv}

	finalPath := KeyPath()
	tmpPath   := finalPath + ".tmp"

	seed  := priv.Seed()
	block := &pem.Block{Type: pemType, Bytes: seed}
	if err := writeKeyFile(tmpPath, pem.EncodeToMemory(block)); err != nil {
		return nil, nil, fmt.Errorf("writing staged key to %s: %w", tmpPath, err)
	}

	return newKP, &PendingRotation{tmpPath: tmpPath, finalPath: finalPath}, nil
}

// Commit atomically replaces the current key file with the staged file.
// After Commit, Load() will return the new key pair.
func (p *PendingRotation) Commit() error {
	if err := os.Rename(p.tmpPath, p.finalPath); err != nil {
		return fmt.Errorf("committing rotated key: %w", err)
	}
	return nil
}

// Rollback deletes the staged key file, leaving the original key unchanged.
func (p *PendingRotation) Rollback() error {
	if err := os.Remove(p.tmpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rolling back staged key: %w", err)
	}
	return nil
}
