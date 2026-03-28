package commander

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/httpsign"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
)

// Task represents a single pending remediation command from the server.
type Task struct {
	ID          int             `json:"id"`
	CommandType string          `json:"command_type"`
	Payload     json.RawMessage `json:"payload"`
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
	return parsed.Data.Commands, nil
}
