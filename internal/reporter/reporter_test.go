package reporter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
)

// testReport returns a minimal DeviceReport for use in tests.
func testReport(hardwareUUID string) *DeviceReport {
	return &DeviceReport{
		SchemaVersion:   "1",
		RegistrationKey: "test-reg-key",
		AgentVersion:    "0.1.3",
		Platform:        "linux",
		DeviceIdentity: DeviceIdentity{
			Hostname:     "test-host",
			HardwareUUID: hardwareUUID,
		},
	}
}

// testConfig returns a minimal Config pointing at the given server URL.
func testConfig(serverURL string) *config.Config {
	cfg := config.Default()
	cfg.RegistrationKey = "test-reg-key"
	cfg.APIEndpoint = serverURL
	cfg.RetryAttempts = 1
	cfg.RetryDelaySeconds = 0
	cfg.HTTPTimeoutSeconds = 5
	return cfg
}

// checkinHandler returns an HTTP handler that responds with a successful
// check-in response containing the given agentID.
func checkinHandler(agentID int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"agent_id":               agentID,
				"issues_found":           0,
				"check_interval_seconds": 14400,
			},
		})
	}
}

// TestSendSetsRegistrationKeyHeader verifies X-Registration-Key is present.
func TestSendSetsRegistrationKeyHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Registration-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":1}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	r := NewWithClient(cfg, srv.Client())
	if err := r.Send(testReport("hw-uuid-001")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if got != "test-reg-key" {
		t.Errorf("X-Registration-Key = %q, want %q", got, "test-reg-key")
	}
}

// TestSendSetsHardwareUUIDHeader verifies X-Hardware-UUID is present and matches.
func TestSendSetsHardwareUUIDHeader(t *testing.T) {
	const wantUUID = "aabbccdd-1122-4000-8000-112233445566"
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Hardware-UUID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":1}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	r := NewWithClient(cfg, srv.Client())
	if err := r.Send(testReport(wantUUID)); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if got != wantUUID {
		t.Errorf("X-Hardware-UUID = %q, want %q", got, wantUUID)
	}
}

// TestSendOmitsAgentIDHeaderWhenNotConfigured ensures X-Agent-ID is absent on first check-in.
func TestSendOmitsAgentIDHeaderWhenNotConfigured(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Agent-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":0}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	// AgentID explicitly empty
	cfg.AgentID = ""
	r := NewWithClient(cfg, srv.Client())
	if err := r.Send(testReport("hw-uuid-002")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if got != "" {
		t.Errorf("X-Agent-ID should be absent on first check-in, got %q", got)
	}
}

// TestSendSetsAgentIDHeaderWhenConfigured ensures X-Agent-ID is sent once the agent has an ID.
func TestSendSetsAgentIDHeaderWhenConfigured(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Agent-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":99}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.AgentID = "99"
	r := NewWithClient(cfg, srv.Client())
	if err := r.Send(testReport("hw-uuid-003")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if got != "99" {
		t.Errorf("X-Agent-ID = %q, want %q", got, "99")
	}
}

// TestSendPersistsAgentIDFromResponse verifies the agent saves agent_id on first enrollment.
func TestSendPersistsAgentIDFromResponse(t *testing.T) {
	// Use a temp config file so Save() doesn't touch the real system path
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("BESTDEFENSE_CONFIG_PATH", configPath)

	srv := httptest.NewServer(checkinHandler(42))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.AgentID = "" // not yet enrolled
	r := NewWithClient(cfg, srv.Client())

	if err := r.Send(testReport("hw-uuid-004")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	// In-memory config should be updated
	if cfg.AgentID != "42" {
		t.Errorf("in-memory AgentID = %q after enrollment, want %q", cfg.AgentID, "42")
	}

	// Config file should also be updated
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() after enrollment: %v", err)
	}
	if loaded.AgentID != "42" {
		t.Errorf("persisted AgentID = %q, want %q", loaded.AgentID, "42")
	}
}

// TestSendDoesNotOverwriteExistingAgentID ensures a re-enrolled agent keeps its ID.
func TestSendDoesNotOverwriteExistingAgentID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("BESTDEFENSE_CONFIG_PATH", configPath)

	// Server returns a different agent_id (shouldn't happen in practice, but guard against it)
	srv := httptest.NewServer(checkinHandler(99))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.AgentID = "42" // already enrolled
	r := NewWithClient(cfg, srv.Client())

	if err := r.Send(testReport("hw-uuid-005")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	// Should NOT overwrite "42" with "99"
	if cfg.AgentID != "42" {
		t.Errorf("AgentID = %q after re-checkin, want original %q", cfg.AgentID, "42")
	}
}

// TestSendRetriesOnTransientFailure verifies retry behaviour.
func TestSendRetriesOnTransientFailure(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":7}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.RetryAttempts = 3
	cfg.RetryDelaySeconds = 0
	r := NewWithClient(cfg, srv.Client())

	if err := r.Send(testReport("hw-uuid-006")); err != nil {
		t.Fatalf("Send() should succeed after retries, got: %v", err)
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", attempt)
	}
}

// TestSendFailsAfterAllRetries verifies the error is surfaced when all retries are exhausted.
func TestSendFailsAfterAllRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.RetryAttempts = 2
	cfg.RetryDelaySeconds = 0
	r := NewWithClient(cfg, srv.Client())

	if err := r.Send(testReport("hw-uuid-007")); err == nil {
		t.Error("Send() should return error when all retries fail")
	}
}

// TestSendSetsTimestampHeaderWhenKeyPairIsSet verifies X-Timestamp is present when signing.
func TestSendSetsTimestampHeaderWhenKeyPairIsSet(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Timestamp")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":1}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	r := NewWithClient(cfg, srv.Client()).WithKeyPair(kp)
	if err := r.Send(testReport("hw-uuid-sign-001")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if got == "" {
		t.Error("X-Timestamp header not set when KeyPair is configured")
	}
}

// TestSendSetsSignatureHeaderWhenKeyPairIsSet verifies X-Signature is present when signing.
func TestSendSetsSignatureHeaderWhenKeyPairIsSet(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Signature")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"agent_id":1}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	r := NewWithClient(cfg, srv.Client()).WithKeyPair(kp)
	if err := r.Send(testReport("hw-uuid-sign-002")); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if got == "" {
		t.Error("X-Signature header not set when KeyPair is configured")
	}
}
