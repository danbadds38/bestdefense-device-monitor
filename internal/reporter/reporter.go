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

// checkinResponse mirrors the JSON envelope returned by POST /agent/checkin.
type checkinResponse struct {
	Success bool            `json:"success"`
	Data    checkinRespData `json:"data"`
}

type checkinRespData struct {
	AgentID              int `json:"agent_id"`
	IssuesFound          int `json:"issues_found"`
	CheckIntervalSeconds int `json:"check_interval_seconds"`
}

// Reporter sends device reports to the BestDefense API.
type Reporter struct {
	cfg    *config.Config
	client *http.Client
}

// New creates a Reporter configured from cfg.
func New(cfg *config.Config) *Reporter {
	return NewWithClient(cfg, &http.Client{
		Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second,
	})
}

// NewWithClient creates a Reporter with an explicit HTTP client (used in tests).
func NewWithClient(cfg *config.Config, client *http.Client) *Reporter {
	return &Reporter{cfg: cfg, client: client}
}

// Send POSTs the report to the configured API endpoint.
// It retries up to cfg.RetryAttempts times on transient errors.
// On the first successful check-in that returns an agent_id, the ID is persisted
// to config.json so all subsequent requests can include X-Agent-ID.
func (r *Reporter) Send(report *DeviceReport) error {
	var lastErr error
	for attempt := 1; attempt <= r.cfg.RetryAttempts; attempt++ {
		resp, err := r.sendOnce(report)
		if err != nil {
			lastErr = err
			if attempt < r.cfg.RetryAttempts {
				time.Sleep(time.Duration(r.cfg.RetryDelaySeconds) * time.Second)
			}
			continue
		}

		// Persist agent_id on first successful enrollment
		if resp != nil && resp.AgentID > 0 && r.cfg.AgentID == "" {
			r.cfg.AgentID = fmt.Sprintf("%d", resp.AgentID)
			// Best-effort save — don't fail the send if we can't persist
			_ = config.Save(r.cfg)
		}
		return nil
	}
	return fmt.Errorf("all %d send attempts failed; last error: %w", r.cfg.RetryAttempts, lastErr)
}

// sendOnce performs a single HTTP POST of the device report and returns the
// parsed server response on success, or an error on failure.
func (r *Reporter) sendOnce(report *DeviceReport) (*checkinRespData, error) {
	body, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshalling report: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.cfg.APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("bestdefense-device-monitor/%s", r.cfg.AgentVersion))
	req.Header.Set("X-Registration-Key", r.cfg.RegistrationKey)
	req.Header.Set("X-Hardware-UUID", report.DeviceIdentity.HardwareUUID)
	if r.cfg.AgentID != "" {
		req.Header.Set("X-Agent-ID", r.cfg.AgentID)
	}

	httpResp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read up to 8KB — enough for the check-in response JSON
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 8192))

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var parsed checkinResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		// Non-fatal: response parse failure does not fail the send
		return nil, nil //nolint:nilerr
	}

	return &parsed.Data, nil
}
