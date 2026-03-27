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
