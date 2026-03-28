package commander

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/httpsign"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
	"github.com/bestdefense/bestdefense-device-monitor/internal/serverkey"
)

// Task represents a single pending remediation command from the server.
type Task struct {
	ID          int             `json:"id"`
	CommandType string          `json:"command_type"`
	Payload     json.RawMessage `json:"payload"`
	IssuedAt    string          `json:"issued_at"`
	Signature   string          `json:"signature"`
}

// commandsResponse mirrors the JSON envelope returned by GET /agent/commands.
type commandsResponse struct {
	Success bool             `json:"success"`
	Data    commandsRespData `json:"data"`
}

type commandsRespData struct {
	Commands []Task `json:"commands"`
}

// Commander polls the server for pending remediation tasks.
type Commander struct {
	cfg    *config.Config
	client *http.Client
	kp     *identity.KeyPair
}

// New creates a Commander configured from cfg.
func New(cfg *config.Config) *Commander {
	return NewWithClient(cfg, &http.Client{
		Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
	})
}

// NewWithClient creates a Commander with an explicit HTTP client (used in tests).
func NewWithClient(cfg *config.Config, client *http.Client) *Commander {
	return &Commander{cfg: cfg, client: client}
}

// WithKeyPair sets the identity key pair used to sign outbound requests.
// Returns the Commander for chaining.
func (c *Commander) WithKeyPair(kp *identity.KeyPair) *Commander {
	c.kp = kp
	return c
}

// Poll fetches pending commands from the server's /agent/commands endpoint.
// Returns an empty slice (not an error) if the agent is not yet enrolled
// (AgentID is empty), since commands require a stable agent identity.
// Commands that fail signature verification are skipped (logged, not returned).
func (c *Commander) Poll() ([]Task, error) {
	if c.cfg.AgentID == "" {
		return []Task{}, nil
	}

	req, err := http.NewRequest(http.MethodGet, c.cfg.CommandsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating commands request: %w", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("bestdefense-device-monitor/%s", c.cfg.AgentVersion))
	req.Header.Set("X-Registration-Key", c.cfg.RegistrationKey)
	req.Header.Set("X-Agent-ID", c.cfg.AgentID)

	if err := httpsign.AddSignature(req, c.kp, nil); err != nil {
		return nil, fmt.Errorf("signing commands request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polling commands: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("commands endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed commandsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing commands response: %w", err)
	}

	if parsed.Data.Commands == nil {
		return []Task{}, nil
	}

	valid := make([]Task, 0, len(parsed.Data.Commands))
	for _, cmd := range parsed.Data.Commands {
		if err := verifyCommandSignature(cmd); err != nil {
			fmt.Printf("WARN: skipping command %d (%s): signature check failed: %v\n", cmd.ID, cmd.CommandType, err)
			continue
		}
		valid = append(valid, cmd)
	}
	return valid, nil
}

// verifyCommandSignature verifies the server's Ed25519 signature on a command.
// Canonical message: "{id}|{command_type}|{issued_at_unix}"
// Returns nil on success, or an error describing the failure.
func verifyCommandSignature(cmd Task) error {
	if cmd.Signature == "" {
		return fmt.Errorf("missing signature")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(cmd.Signature)
	if err != nil {
		return fmt.Errorf("base64 decode signature: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("signature must be %d bytes, got %d", ed25519.SignatureSize, len(sigBytes))
	}

	if cmd.IssuedAt == "" {
		return fmt.Errorf("missing issued_at")
	}

	issuedAt, err := time.Parse(time.RFC3339, cmd.IssuedAt)
	if err != nil {
		return fmt.Errorf("parse issued_at %q: %w", cmd.IssuedAt, err)
	}

	if time.Since(issuedAt) > 24*time.Hour {
		return fmt.Errorf("command is stale (issued %s)", cmd.IssuedAt)
	}

	msg := fmt.Sprintf("%d|%s|%d", cmd.ID, cmd.CommandType, issuedAt.Unix())
	if !ed25519.Verify(serverkey.PublicKey(), []byte(msg), sigBytes) {
		return fmt.Errorf("Ed25519 signature verification failed")
	}

	return nil
}
