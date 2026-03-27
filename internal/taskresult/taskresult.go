package taskresult

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/executor"
)

// Poster sends task execution results to the server's /agent/task-result endpoint.
type Poster struct {
	cfg    *config.Config
	client *http.Client
}

// New creates a Poster configured from cfg with a default HTTP client.
func New(cfg *config.Config) *Poster {
	return NewWithClient(cfg, &http.Client{
		Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
	})
}

// NewWithClient creates a Poster with an explicit HTTP client (used in tests).
func NewWithClient(cfg *config.Config, client *http.Client) *Poster {
	return &Poster{cfg: cfg, client: client}
}

type resultPayload struct {
	TaskID     int    `json:"task_id"`
	Status     string `json:"status"`
	Output     string `json:"output"`
	ExecutedAt string `json:"executed_at"`
}

// Post sends each result to the server individually.
// All requests are attempted; errors are collected and returned as a single error.
// Returns nil if there are no results or all posts succeed.
func (p *Poster) Post(results []executor.Result) error {
	if len(results) == 0 {
		return nil
	}

	var errs []string
	for _, r := range results {
		if err := p.postOne(r); err != nil {
			errs = append(errs, fmt.Sprintf("task %d: %v", r.TaskID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("task result errors: %v", errs)
	}
	return nil
}

func (p *Poster) postOne(r executor.Result) error {
	payload := resultPayload{
		TaskID:     r.TaskID,
		Status:     r.Status,
		Output:     r.Output,
		ExecutedAt: r.ExecutedAt.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling result: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.cfg.TaskResultEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("bestdefense-device-monitor/%s", p.cfg.AgentVersion))
	req.Header.Set("X-Registration-Key", p.cfg.RegistrationKey)
	req.Header.Set("X-Agent-ID", p.cfg.AgentID)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending result: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
