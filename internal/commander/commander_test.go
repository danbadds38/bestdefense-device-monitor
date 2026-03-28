package commander

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
	"github.com/bestdefense/bestdefense-device-monitor/internal/serverkey"
)

// Test key: seed = 0x43 * 32.
// The private key is intentionally committed — test-only key.
// Public key (base64): Ivwpd5Lwtv/Av8/bftsMCqFOAlo2XsDjQuhuOCnLdLY=
func testPrivKey() ed25519.PrivateKey {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = 0x43
	}
	return ed25519.NewKeyFromSeed(seed)
}

// signCmd signs a command using the test private key.
// canonical message: "{id}|{command_type}|{issued_at_unix}"
func signCmd(id int, commandType string, issuedAt time.Time) string {
	msg := fmt.Sprintf("%d|%s|%d", id, commandType, issuedAt.Unix())
	sig := ed25519.Sign(testPrivKey(), []byte(msg))
	return base64.StdEncoding.EncodeToString(sig)
}

// signedTask returns a Task with a valid server signature using the test key.
func signedTask(id int, commandType string) Task {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	return Task{
		ID:          id,
		CommandType: commandType,
		IssuedAt:    issuedAt.Format(time.RFC3339),
		Signature:   signCmd(id, commandType, issuedAt),
	}
}

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
	// Tasks must be signed with the test key; unsigned tasks are filtered out.
	want := []Task{
		signedTask(7, "enable_firewall"),
		signedTask(8, "enable_screen_lock"),
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

// TestPollSetsTimestampHeaderWhenKeyPairIsSet verifies X-Timestamp is present when signing.
func TestPollSetsTimestampHeaderWhenKeyPairIsSet(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Timestamp")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"commands":[]}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client()).WithKeyPair(kp)
	if _, err := c.Poll(); err != nil {
		t.Fatalf("Poll() error: %v", err)
	}

	if got == "" {
		t.Error("X-Timestamp header not set when KeyPair is configured")
	}
}

// TestPollSetsSignatureHeaderWhenKeyPairIsSet verifies X-Signature is present when signing.
func TestPollSetsSignatureHeaderWhenKeyPairIsSet(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/test.key")
	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Signature")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"success":true,"data":{"commands":[]}}`)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client()).WithKeyPair(kp)
	if _, err := c.Poll(); err != nil {
		t.Fatalf("Poll() error: %v", err)
	}

	if got == "" {
		t.Error("X-Signature header not set when KeyPair is configured")
	}
}

// TestPollSkipsCommandWithMissingSignature ensures unsigned commands are filtered.
func TestPollSkipsCommandWithMissingSignature(t *testing.T) {
	tasks := []Task{
		{ID: 1, CommandType: "enable_firewall", IssuedAt: time.Now().UTC().Format(time.RFC3339)},
	}
	srv := httptest.NewServer(commandsHandler(tasks))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	got, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Poll() returned %d tasks, want 0 (unsigned command must be skipped)", len(got))
	}
}

// TestPollSkipsCommandWithInvalidSignature ensures commands with bad signatures are filtered.
func TestPollSkipsCommandWithInvalidSignature(t *testing.T) {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	tasks := []Task{
		{
			ID:          2,
			CommandType: "enable_screen_lock",
			IssuedAt:    issuedAt.Format(time.RFC3339),
			Signature:   base64.StdEncoding.EncodeToString(make([]byte, 64)), // 64 zero bytes
		},
	}
	srv := httptest.NewServer(commandsHandler(tasks))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	got, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Poll() returned %d tasks, want 0 (invalid signature must be skipped)", len(got))
	}
}

// TestPollSkipsStaleCommand ensures commands issued more than 24 hours ago are filtered.
func TestPollSkipsStaleCommand(t *testing.T) {
	staleTime := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Second)
	tasks := []Task{
		{
			ID:          3,
			CommandType: "enable_firewall",
			IssuedAt:    staleTime.Format(time.RFC3339),
			Signature:   signCmd(3, "enable_firewall", staleTime),
		},
	}
	srv := httptest.NewServer(commandsHandler(tasks))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	got, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Poll() returned %d tasks, want 0 (stale command must be skipped)", len(got))
	}
}

// TestPollAcceptsValidSignedCommand verifies that a correctly signed command passes.
func TestPollAcceptsValidSignedCommand(t *testing.T) {
	task := signedTask(4, "enable_firewall")
	srv := httptest.NewServer(commandsHandler([]Task{task}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	got, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Poll() returned %d tasks, want 1", len(got))
	}
	if got[0].ID != task.ID {
		t.Errorf("task ID = %d, want %d", got[0].ID, task.ID)
	}
}

// TestPollReturnsOnlyValidCommandsFromMixedList verifies that only signed commands pass.
func TestPollReturnsOnlyValidCommandsFromMixedList(t *testing.T) {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	tasks := []Task{
		signedTask(10, "enable_firewall"),
		// unsigned
		{ID: 11, CommandType: "enable_screen_lock", IssuedAt: issuedAt.Format(time.RFC3339)},
		signedTask(12, "request_reboot"),
		// bad sig — 64 zero bytes
		{ID: 13, CommandType: "enable_auto_updates", IssuedAt: issuedAt.Format(time.RFC3339),
			Signature: base64.StdEncoding.EncodeToString(make([]byte, 64))},
	}

	srv := httptest.NewServer(commandsHandler(tasks))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	c := NewWithClient(cfg, srv.Client())
	got, err := c.Poll()
	if err != nil {
		t.Fatalf("Poll() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Poll() returned %d tasks, want 2 (only valid signed commands)", len(got))
	}
	if got[0].ID != 10 || got[1].ID != 12 {
		t.Errorf("unexpected task IDs: %d, %d", got[0].ID, got[1].ID)
	}
}

// TestServerKeyPublicKeyMatchesTestPrivateKey verifies the embedded test key round-trips.
func TestServerKeyPublicKeyMatchesTestPrivateKey(t *testing.T) {
	pubKey := serverkey.PublicKey()
	priv := testPrivKey()
	msg := []byte("test-message")
	sig := ed25519.Sign(priv, msg)
	if !ed25519.Verify(pubKey, msg, sig) {
		t.Error("serverkey.PublicKey() does not match the test private key")
	}
}
