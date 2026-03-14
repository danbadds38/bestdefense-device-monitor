package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
)

// Reporter sends device reports to the BestDefense API.
type Reporter struct {
	cfg    *config.Config
	client *http.Client
}

// New creates a Reporter configured from cfg.
func New(cfg *config.Config) *Reporter {
	return &Reporter{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
		},
	}
}

// Send POSTs the report to the configured API endpoint.
// It retries up to cfg.RetryAttempts times on transient errors.
func (r *Reporter) Send(report *DeviceReport) error {
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshalling report: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= r.cfg.RetryAttempts; attempt++ {
		if err := r.sendOnce(body); err != nil {
			lastErr = err
			if attempt < r.cfg.RetryAttempts {
				time.Sleep(time.Duration(r.cfg.RetryDelaySeconds) * time.Second)
			}
			continue
		}
		return nil // success
	}
	return fmt.Errorf("all %d send attempts failed; last error: %w", r.cfg.RetryAttempts, lastErr)
}

func (r *Reporter) sendOnce(body []byte) error {
	req, err := http.NewRequest(http.MethodPost, r.cfg.APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("bestdefense-device-monitor/%s", r.cfg.AgentVersion))
	req.Header.Set("X-Registration-Key", r.cfg.RegistrationKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read body for error messages (limit to 4KB)
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
