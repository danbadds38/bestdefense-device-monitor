package keyrotation

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/httpsign"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
)

// Rotator posts a newly generated public key to the server and atomically commits
// the new key pair to disk on success.
type Rotator struct {
	cfg    *config.Config
	client *http.Client
}

// New creates a Rotator configured from cfg.
func New(cfg *config.Config) *Rotator {
	return NewWithClient(cfg, &http.Client{
		Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
	})
}

// NewWithClient creates a Rotator with an explicit HTTP client (used in tests).
func NewWithClient(cfg *config.Config, client *http.Client) *Rotator {
	return &Rotator{cfg: cfg, client: client}
}

// Rotate generates a new Ed25519 key pair, posts the new public key to the server
// (signed with the OLD private key so the server can verify it), and commits the
// new key to disk on a successful 2xx response.
//
// On any error the staged key file is rolled back, leaving the old key intact.
// Returns the new KeyPair on success.
func (r *Rotator) Rotate(oldKP *identity.KeyPair) (*identity.KeyPair, error) {
	newKP, pending, err := identity.Rotate()
	if err != nil {
		return nil, fmt.Errorf("generating new key pair: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.cfg.RotateKeyEndpoint, nil)
	if err != nil {
		_ = pending.Rollback()
		return nil, fmt.Errorf("creating rotation request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("bestdefense-device-monitor/%s", r.cfg.AgentVersion))
	req.Header.Set("X-Registration-Key", r.cfg.RegistrationKey)
	req.Header.Set("X-Agent-ID", r.cfg.AgentID)
	req.Header.Set("X-Agent-Public-Key", newKP.PublicKeyBase64())

	// Sign with the OLD key — the server still holds the old public key for verification.
	if err := httpsign.AddSignature(req, oldKP, nil); err != nil {
		_ = pending.Rollback()
		return nil, fmt.Errorf("signing rotation request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		_ = pending.Rollback()
		return nil, fmt.Errorf("posting rotation request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 1024)) // drain body

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = pending.Rollback()
		return nil, fmt.Errorf("server returned status %d for key rotation", resp.StatusCode)
	}

	if err := pending.Commit(); err != nil {
		return nil, fmt.Errorf("committing new key to disk: %w", err)
	}

	return newKP, nil
}
