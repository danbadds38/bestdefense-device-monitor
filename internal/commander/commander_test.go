package commander

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
)

// testConfig returns a minimal Config pointing at the given server URL.
func testConfig(serverURL string) *config.Config {
	cfg := config.Default()
	cfg.RegistrationKey = "test-reg-key"
	cfg.AgentID = "42"
	cfg.CommandsEndpoint = serverURL
	cfg.HTTPTimeoutSeconds = 5
	return cfg
}

// commandsHandler returns an HTTP handler that responds with the given tasks.
func commandsHandler(tasks []Task) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"commands": tasks,
			},
		})
	}
}

func TestPollSetsRegistrationKeyHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Registration-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"commands":[]}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	if _, err := c.Poll(); err != nil {
		t.Fatalf("Poll() error: %v", err)
	}

	if got != "test-reg-key" {
		t.Errorf("X-Registration-Key = %q, want %q", got, "test-reg-key")
	}
}

func TestPollSetsAgentIDHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Agent-ID")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"commands":[]}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.AgentID = "99"
	c := NewWithClient(cfg, srv.Client())
	if _, err := c.Poll(); err != nil {
		t.Fatalf("Poll() error: %v", err)
	}

	if got != "99" {
		t.Errorf("X-Agent-ID = %q, want %q", got, "99")
	}
}

func TestPollReturnsTasksFromResponse(t *testing.T) {
	want := []Task{
		{ID: 7, CommandType: "enable_firewall"},
		{ID: 8, CommandType: "enable_screen_lock"},
	}

	srv := httptest.NewServer(commandsHandler(want))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	got, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("Poll() returned %d tasks, want %d", len(got), len(want))
	}
	for i, task := range got {
		if task.ID != want[i].ID {
			t.Errorf("task[%d].ID = %d, want %d", i, task.ID, want[i].ID)
		}
		if task.CommandType != want[i].CommandType {
			t.Errorf("task[%d].CommandType = %q, want %q", i, task.CommandType, want[i].CommandType)
		}
	}
}

func TestPollReturnsEmptySliceWhenNoTasks(t *testing.T) {
	srv := httptest.NewServer(commandsHandler([]Task{}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	tasks, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Poll() returned %d tasks, want 0", len(tasks))
	}
}

func TestPollReturnsEmptySliceWhenAgentIDNotSet(t *testing.T) {
	// Server should never be called when AgentID is empty
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.AgentID = "" // not yet enrolled
	c := NewWithClient(cfg, srv.Client())

	tasks, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() should not error when AgentID is empty, got: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Poll() returned %d tasks, want 0 (unenrolled)", len(tasks))
	}
	if called {
		t.Error("server should not be called when AgentID is empty")
	}
}

func TestPollReturnsErrorOnNetworkFailure(t *testing.T) {
	cfg := testConfig("http://127.0.0.1:1") // no server listening
	cfg.HTTPTimeoutSeconds = 1
	c := New(cfg) // use real client so it actually fails to connect
	_, err := c.Poll()
	if err == nil {
		t.Error("Poll() should return error on connection failure")
	}
}

func TestPollReturnsErrorOnNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"message":"bad key"}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	_, err := c.Poll()
	if err == nil {
		t.Error("Poll() should return error for 4xx response")
	}
}
