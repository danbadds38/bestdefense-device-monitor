package commander

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// signScriptCmd signs an execute_script command using the test private key.
// Extended canonical message: "{id}|execute_script|{issued_at_unix}|{script_sha256_hex}"
func signScriptCmd(id int, issuedAt time.Time, scriptHashHex string) string {
	msg := fmt.Sprintf("%d|execute_script|%d|%s", id, issuedAt.Unix(), scriptHashHex)
	sig := ed25519.Sign(testPrivKey(), []byte(msg))
	return base64.StdEncoding.EncodeToString(sig)
}

// scriptHashHex returns the hex-encoded SHA-256 of content.
func scriptHashHex(content string) string {
	d := sha256.Sum256([]byte(content))
	return hex.EncodeToString(d[:])
}

// validScriptTask builds a fully-signed, hash-consistent execute_script Task.
func validScriptTask(id int, content string) Task {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	hashHex := scriptHashHex(content)
	payload, _ := json.Marshal(ExecuteScriptPayload{
		ScriptContent:  content,
		ScriptHash:     hashHex,
		DryRun:         false,
		DispatchID:     1,
		TimeoutSeconds: 60,
	})
	return Task{
		ID:          id,
		CommandType: "execute_script",
		Payload:     json.RawMessage(payload),
		IssuedAt:    issuedAt.Format(time.RFC3339),
		Signature:   signScriptCmd(id, issuedAt, hashHex),
	}
}

// TestVerifyExecuteScriptAcceptsValidTask verifies that a correctly signed
// execute_script task with a matching content hash passes verification.
func TestVerifyExecuteScriptAcceptsValidTask(t *testing.T) {
	task := validScriptTask(100, "echo hello")
	if err := verifyCommandSignature(task); err != nil {
		t.Errorf("verifyCommandSignature() returned unexpected error: %v", err)
	}
}

// TestVerifyExecuteScriptRejectsMissingHash verifies that an execute_script
// payload with an empty script_hash is rejected before the signature check.
func TestVerifyExecuteScriptRejectsMissingHash(t *testing.T) {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	payload, _ := json.Marshal(ExecuteScriptPayload{
		ScriptContent:  "echo hello",
		ScriptHash:     "", // deliberately empty
		TimeoutSeconds: 30,
	})
	// Sign with the real hash so the signature itself would be valid if hash were present.
	realHash := scriptHashHex("echo hello")
	task := Task{
		ID:          101,
		CommandType: "execute_script",
		Payload:     json.RawMessage(payload),
		IssuedAt:    issuedAt.Format(time.RFC3339),
		Signature:   signScriptCmd(101, issuedAt, realHash),
	}
	err := verifyCommandSignature(task)
	if err == nil {
		t.Fatal("verifyCommandSignature() should return error for missing script_hash")
	}
}

// TestVerifyExecuteScriptRejectsMalformedPayload verifies that an execute_script
// command with a payload that cannot be unmarshalled is rejected.
func TestVerifyExecuteScriptRejectsMalformedPayload(t *testing.T) {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	task := Task{
		ID:          102,
		CommandType: "execute_script",
		Payload:     json.RawMessage(`{not valid json`),
		IssuedAt:    issuedAt.Format(time.RFC3339),
		Signature:   signScriptCmd(102, issuedAt, "deadbeef"),
	}
	err := verifyCommandSignature(task)
	if err == nil {
		t.Fatal("verifyCommandSignature() should return error for malformed payload")
	}
}

// TestVerifyExecuteScriptRejectsTamperedContent verifies that when script_content
// is changed after signing, verifyCommandSignature returns ErrTamperDetected.
// The signature covers the original hash, so it will still pass the signature
// check — but the re-hash of the modified content won't match.
func TestVerifyExecuteScriptRejectsTamperedContent(t *testing.T) {
	issuedAt := time.Now().UTC().Truncate(time.Second)
	originalContent := "echo original"
	hashHex := scriptHashHex(originalContent)

	// Build task signed over the original content hash…
	payload, _ := json.Marshal(ExecuteScriptPayload{
		ScriptContent:  "echo TAMPERED", // content changed after signing
		ScriptHash:     hashHex,         // hash still matches original
		TimeoutSeconds: 30,
	})
	task := Task{
		ID:          103,
		CommandType: "execute_script",
		Payload:     json.RawMessage(payload),
		IssuedAt:    issuedAt.Format(time.RFC3339),
		Signature:   signScriptCmd(103, issuedAt, hashHex),
	}

	err := verifyCommandSignature(task)
	if !errors.Is(err, ErrTamperDetected) {
		t.Errorf("expected ErrTamperDetected, got: %v", err)
	}
}

// TestPollAcceptsValidExecuteScriptCommand verifies that a correctly signed
// execute_script command passes end-to-end through Poll().
func TestPollAcceptsValidExecuteScriptCommand(t *testing.T) {
	task := validScriptTask(200, "Get-Process")
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

	// Verify the payload round-trips correctly.
	var p ExecuteScriptPayload
	if err := json.Unmarshal(got[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if p.ScriptContent != "Get-Process" {
		t.Errorf("ScriptContent = %q, want %q", p.ScriptContent, "Get-Process")
	}
}
