package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
)

// scriptHashHex returns the hex-encoded SHA-256 of content.
func scriptHashHex(content string) string {
	d := sha256.Sum256([]byte(content))
	return hex.EncodeToString(d[:])
}

// makeScriptTask builds a commander.Task with a properly formatted
// execute_script payload.  The ScriptHash always matches ScriptContent so the
// defense-in-depth hash check in handleExecuteScript passes (unless the caller
// deliberately corrupts the hash to test tamper detection).
func makeScriptTask(id int, content string, dryRun bool, dispatchID int) commander.Task {
	hashHex := scriptHashHex(content)
	payload, _ := json.Marshal(commander.ExecuteScriptPayload{
		ScriptContent:  content,
		ScriptHash:     hashHex,
		DryRun:         dryRun,
		DispatchID:     dispatchID,
		TimeoutSeconds: 30,
	})
	return commander.Task{
		ID:          id,
		CommandType: "execute_script",
		Payload:     json.RawMessage(payload),
	}
}

// makeTamperedScriptTask builds a task where the ScriptHash deliberately does
// NOT match the ScriptContent, simulating in-memory tampering.
func makeTamperedScriptTask(id int) commander.Task {
	payload, _ := json.Marshal(commander.ExecuteScriptPayload{
		ScriptContent:  "echo tampered",
		ScriptHash:     "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		TimeoutSeconds: 30,
	})
	return commander.Task{
		ID:          id,
		CommandType: "execute_script",
		Payload:     json.RawMessage(payload),
	}
}

// TestRunExecuteScriptSuccess verifies that a valid execute_script task is
// executed and returns a success result with the script output.
func TestRunExecuteScriptSuccess(t *testing.T) {
	task := makeScriptTask(50, "echo hello_exec", false, 99)
	results, _ := Run([]commander.Task{task}, nil)

	if len(results) != 1 {
		t.Fatalf("Run() returned %d results, want 1", len(results))
	}
	r := results[0]
	if r.Status != "success" {
		t.Errorf("Status = %q, want %q", r.Status, "success")
	}
	if !strings.Contains(r.Output, "hello_exec") {
		t.Errorf("Output = %q, want it to contain %q", r.Output, "hello_exec")
	}
	if r.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", r.ExitCode)
	}
	if r.DispatchID != 99 {
		t.Errorf("DispatchID = %d, want 99", r.DispatchID)
	}
	if r.TamperDetected {
		t.Error("TamperDetected should be false for a valid task")
	}
}

// TestRunExecuteScriptTamperDetected verifies that a task whose script content
// does not match the signed hash is rejected with TamperDetected=true.
func TestRunExecuteScriptTamperDetected(t *testing.T) {
	task := makeTamperedScriptTask(51)
	results, _ := Run([]commander.Task{task}, nil)

	if len(results) != 1 {
		t.Fatalf("Run() returned %d results, want 1", len(results))
	}
	r := results[0]
	if r.Status != "failed" {
		t.Errorf("Status = %q, want %q", r.Status, "failed")
	}
	if !r.TamperDetected {
		t.Error("TamperDetected should be true when content hash mismatches")
	}
}

// TestRunExecuteScriptDryRunSetsDryRunDiff verifies that when dry_run=true the
// script output is stored in DryRunDiff and Output is empty.
func TestRunExecuteScriptDryRunSetsDryRunDiff(t *testing.T) {
	task := makeScriptTask(52, "echo dry_run_output", true, 42)
	results, _ := Run([]commander.Task{task}, nil)

	if len(results) != 1 {
		t.Fatalf("Run() returned %d results, want 1", len(results))
	}
	r := results[0]
	if r.Status != "success" {
		t.Errorf("Status = %q, want %q", r.Status, "success")
	}
	if !strings.Contains(r.DryRunDiff, "dry_run_output") {
		t.Errorf("DryRunDiff = %q, want it to contain %q", r.DryRunDiff, "dry_run_output")
	}
	if r.Output != "" {
		t.Errorf("Output = %q, want empty for dry-run", r.Output)
	}
}

// TestRunExecuteScriptNonZeroExitCodeFails verifies that a non-zero exit code
// results in Status="failed" and ExitCode is preserved.
func TestRunExecuteScriptNonZeroExitCodeFails(t *testing.T) {
	task := makeScriptTask(53, "exit 7", false, 0)
	results, _ := Run([]commander.Task{task}, nil)

	if len(results) != 1 {
		t.Fatalf("Run() returned %d results, want 1", len(results))
	}
	r := results[0]
	if r.Status != "failed" {
		t.Errorf("Status = %q, want %q", r.Status, "failed")
	}
	if r.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", r.ExitCode)
	}
}

// TestRunShortPollTrueWhenExecuteScriptPresent verifies that Run returns
// shortPoll=true when at least one execute_script task is in the input.
func TestRunShortPollTrueWhenExecuteScriptPresent(t *testing.T) {
	tasks := []commander.Task{
		makeScriptTask(60, "echo ok", false, 0),
	}
	_, shortPoll := Run(tasks, nil)
	if !shortPoll {
		t.Error("shortPoll should be true when execute_script task is present")
	}
}

// TestRunShortPollFalseWithoutExecuteScript verifies that Run returns
// shortPoll=false when no execute_script tasks are present.
func TestRunShortPollFalseWithoutExecuteScript(t *testing.T) {
	withFirewall(func() (string, error) { return "ok", nil }, func() {
		tasks := []commander.Task{
			{ID: 70, CommandType: "enable_firewall"},
		}
		_, shortPoll := Run(tasks, nil)
		if shortPoll {
			t.Error("shortPoll should be false when no execute_script tasks are present")
		}
	})
}
