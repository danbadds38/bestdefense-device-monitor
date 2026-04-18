package taskresult

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/executor"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
)

func testConfig(serverURL string) *config.Config {
	cfg := config.Default()
	cfg.RegistrationKey    = "test-reg-key"
	cfg.AgentID            = "77"
	cfg.TaskResultEndpoint = serverURL
	cfg.HTTPTimeoutSeconds = 5
	return cfg
}

func singleResult() []executor.Result {
	return []executor.Result{
		{
			TaskID:      7,
			CommandType: "enable_firewall",
			Status:      "success",
			Output:      "Firewall enabled.",
			ExecutedAt:  time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC),
		},
	}
}

func TestPostSendsRegistrationKeyHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Registration-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	if err := p.Post(singleResult()); err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if got != "test-reg-key" {
		t.Errorf("X-Registration-Key = %q, want %q", got, "test-reg-key")
	}
}

func TestPostSendsAgentIDHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Agent-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	if err := p.Post(singleResult()); err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if got != "77" {
		t.Errorf("X-Agent-ID = %q, want %q", got, "77")
	}
}

func TestPostSendsCorrectBody(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	results := []executor.Result{{
		TaskID:      7,
		CommandType: "enable_firewall",
		Status:      "success",
		Output:      "Firewall enabled.",
		ExecutedAt:  time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC),
	}}

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	if err := p.Post(results); err != nil {
		t.Fatalf("Post() error: %v", err)
	}

	if int(body["task_id"].(float64)) != 7 {
		t.Errorf("task_id = %v, want 7", body["task_id"])
	}
	if body["status"] != "success" {
		t.Errorf("status = %v, want %q", body["status"], "success")
	}
	if body["output"] != "Firewall enabled." {
		t.Errorf("output = %v, want %q", body["output"], "Firewall enabled.")
	}
	if body["executed_at"] != "2026-03-27T10:00:00Z" {
		t.Errorf("executed_at = %v, want %q", body["executed_at"], "2026-03-27T10:00:00Z")
	}
}

func TestPostReturnsErrorOnNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"message":"bad key"}`)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	err := p.Post(singleResult())
	if err == nil {
		t.Error("Post() should return error for 4xx response")
	}
}

func TestPostReturnsErrorOnNetworkFailure(t *testing.T) {
	cfg := testConfig("http://127.0.0.1:1") // no server
	cfg.HTTPTimeoutSeconds = 1
	p := New(cfg)
	err := p.Post(singleResult())
	if err == nil {
		t.Error("Post() should return error on connection failure")
	}
}

func TestPostIsNoOpWhenNoResults(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	if err := p.Post([]executor.Result{}); err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if called {
		t.Error("server should not be called for empty results")
	}
}

// TestPostSetsTimestampHeaderWhenKeyPairIsSet verifies X-Timestamp is present when signing.
func TestPostSetsTimestampHeaderWhenKeyPairIsSet(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Timestamp")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client()).WithKeyPair(kp)
	if err := p.Post(singleResult()); err != nil {
		t.Fatalf("Post() error: %v", err)
	}

	if got == "" {
		t.Error("X-Timestamp header not set when KeyPair is configured")
	}
}

// TestPostSendsExecuteScriptFields verifies that execute_script-specific fields
// (exit_code, dry_run_diff, dispatch_id, tamper_detected) are included in the
// JSON body when they carry non-zero values.
func TestPostSendsExecuteScriptFields(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	results := []executor.Result{{
		TaskID:         10,
		CommandType:    "execute_script",
		Status:         "failed",
		Output:         "",
		DryRunDiff:     "would change firewall rules",
		ExitCode:       1,
		DispatchID:     42,
		TamperDetected: true,
		ExecutedAt:     time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC),
	}}

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	if err := p.Post(results); err != nil {
		t.Fatalf("Post() error: %v", err)
	}

	if int(body["exit_code"].(float64)) != 1 {
		t.Errorf("exit_code = %v, want 1", body["exit_code"])
	}
	if body["dry_run_diff"] != "would change firewall rules" {
		t.Errorf("dry_run_diff = %v, want %q", body["dry_run_diff"], "would change firewall rules")
	}
	if int(body["dispatch_id"].(float64)) != 42 {
		t.Errorf("dispatch_id = %v, want 42", body["dispatch_id"])
	}
	if body["tamper_detected"] != true {
		t.Errorf("tamper_detected = %v, want true", body["tamper_detected"])
	}
}

// TestPostOmitsExecuteScriptFieldsWhenZero verifies that zero-value
// execute_script fields are omitted from the JSON body for non-script tasks.
func TestPostOmitsExecuteScriptFieldsWhenZero(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client())
	if err := p.Post(singleResult()); err != nil {
		t.Fatalf("Post() error: %v", err)
	}

	if _, ok := body["dispatch_id"]; ok {
		t.Error("dispatch_id should be omitted when zero")
	}
	if _, ok := body["tamper_detected"]; ok {
		t.Error("tamper_detected should be omitted when false")
	}
	if _, ok := body["dry_run_diff"]; ok {
		t.Error("dry_run_diff should be omitted when empty")
	}
}

// TestPostSetsSignatureHeaderWhenKeyPairIsSet verifies X-Signature is present when signing.
func TestPostSetsSignatureHeaderWhenKeyPairIsSet(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Signature")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true}`)
	}))
	defer srv.Close()

	p := NewWithClient(testConfig(srv.URL), srv.Client()).WithKeyPair(kp)
	if err := p.Post(singleResult()); err != nil {
		t.Fatalf("Post() error: %v", err)
	}

	if got == "" {
		t.Error("X-Signature header not set when KeyPair is configured")
	}
}
