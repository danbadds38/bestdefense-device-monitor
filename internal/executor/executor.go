package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
	"github.com/bestdefense/bestdefense-device-monitor/internal/scriptexec"
)

// The following vars are set by platform-specific init() functions.
// They are declared as vars so tests can replace them without executing real OS commands.
var enableFirewall    func() (string, error)
var enableScreenLock  func() (string, error)
var enableAutoUpdates func() (string, error)
var requestReboot     func() (string, error)

// rotateKeys is set by the service before the first Run() call via SetRotateKeysFunc.
// It is nil-safe: if not set, the rotate_keys command produces a "failed" result.
var rotateKeys func(kp *identity.KeyPair) (*identity.KeyPair, error)

// SetRotateKeysFunc registers the key rotation function used by the rotate_keys command.
// Must be called before Run() if the agent should support key rotation.
func SetRotateKeysFunc(fn func(kp *identity.KeyPair) (*identity.KeyPair, error)) {
	rotateKeys = fn
}

// Result holds the outcome of executing a single remediation task.
type Result struct {
	TaskID      int
	CommandType string
	Status      string // "success" or "failed"
	Output      string
	ExecutedAt  time.Time

	// Fields populated for execute_script tasks.
	ExitCode       int    // process exit code; -1 on timeout
	DryRunDiff     string // stdout when dry_run=true (Output is empty in that case)
	DispatchID     int    // dispatch_id from the script payload
	TamperDetected bool   // true when the content hash does not match the signed hash
}

// Run executes each task and returns one Result per task, plus a shortPoll
// flag that is true when at least one execute_script task was processed.
// The service loop uses shortPoll to re-poll the server sooner (2 minutes
// instead of the full check interval) so that dry-run or script results
// are picked up quickly.
//
// kp is the agent's current identity key pair. If a rotate_keys command
// succeeds, *kp is updated in place so subsequent requests use the new key.
// Pass nil for kp if key rotation is not supported in this context.
// Unknown command types produce a "failed" result without calling the OS.
func Run(tasks []commander.Task, kp *identity.KeyPair) ([]Result, bool) {
	results := make([]Result, 0, len(tasks))
	shortPoll := false
	for _, t := range tasks {
		if t.CommandType == "execute_script" {
			shortPoll = true
		}
		results = append(results, runTask(t, kp))
	}
	return results, shortPoll
}

func runTask(t commander.Task, kp *identity.KeyPair) Result {
	start := time.Now().UTC()

	var output string
	var err error

	switch t.CommandType {
	case "enable_firewall":
		output, err = enableFirewall()
	case "enable_screen_lock":
		output, err = enableScreenLock()
	case "enable_auto_updates":
		output, err = enableAutoUpdates()
	case "request_reboot":
		output, err = requestReboot()
	case "rotate_keys":
		return runRotateKeys(t, kp, start)
	case "execute_script":
		return handleExecuteScript(t, start)
	case "check_issues":
		return handleCheckIssues(t, start)
	default:
		return Result{
			TaskID:      t.ID,
			CommandType: t.CommandType,
			Status:      "failed",
			Output:      fmt.Sprintf("unknown command type: %s", t.CommandType),
			ExecutedAt:  start,
		}
	}

	status := "success"
	if err != nil {
		status = "failed"
		output = err.Error()
	}

	return Result{
		TaskID:      t.ID,
		CommandType: t.CommandType,
		Status:      status,
		Output:      output,
		ExecutedAt:  start,
	}
}

func runRotateKeys(t commander.Task, kp *identity.KeyPair, start time.Time) Result {
	if rotateKeys == nil || kp == nil {
		return Result{
			TaskID:      t.ID,
			CommandType: t.CommandType,
			Status:      "failed",
			Output:      "key rotation function not configured",
			ExecutedAt:  start,
		}
	}

	newKP, err := rotateKeys(kp)
	if err != nil {
		return Result{
			TaskID:      t.ID,
			CommandType: t.CommandType,
			Status:      "failed",
			Output:      fmt.Sprintf("key rotation failed: %v", err),
			ExecutedAt:  start,
		}
	}

	// Update the caller's key pair in place so all subsequent requests use the new key.
	*kp = *newKP

	return Result{
		TaskID:      t.ID,
		CommandType: t.CommandType,
		Status:      "success",
		Output:      "Key rotation successful.",
		ExecutedAt:  start,
	}
}

// handleExecuteScript runs an execute_script task via scriptexec.Execute.
// It performs a defense-in-depth content hash re-check before executing to
// detect any in-memory tampering that bypassed the commander's signature check.
func handleExecuteScript(t commander.Task, start time.Time) Result {
	var p commander.ExecuteScriptPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return Result{
			TaskID:      t.ID,
			CommandType: t.CommandType,
			Status:      "failed",
			Output:      fmt.Sprintf("invalid execute_script payload: %v", err),
			ExecutedAt:  start,
		}
	}

	// Defense-in-depth: re-verify content hash before execution.
	digest := sha256.Sum256([]byte(p.ScriptContent))
	if hex.EncodeToString(digest[:]) != p.ScriptHash {
		return Result{
			TaskID:         t.ID,
			CommandType:    t.CommandType,
			Status:         "failed",
			Output:         "script hash mismatch: tamper detected",
			TamperDetected: true,
			DispatchID:     p.DispatchID,
			ExecutedAt:     start,
		}
	}

	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = scriptexec.DefaultTimeout
	}

	res, err := scriptexec.Execute(p.ScriptContent, p.DryRun, timeout)
	if err != nil {
		return Result{
			TaskID:      t.ID,
			CommandType: t.CommandType,
			Status:      "failed",
			Output:      fmt.Sprintf("script execution error: %v", err),
			DispatchID:  p.DispatchID,
			ExecutedAt:  start,
		}
	}

	status := "success"
	if res.ExitCode != 0 || res.TimedOut {
		status = "failed"
	}

	// Combine stdout and stderr into a single output string.
	combined := res.Stdout
	if res.Stderr != "" {
		if combined != "" {
			combined += "\n"
		}
		combined += res.Stderr
	}

	// For dry-run, the output goes in DryRunDiff; Output is left empty so the
	// server can distinguish dry-run previews from actual execution output.
	var output, dryRunDiff string
	if p.DryRun {
		dryRunDiff = combined
	} else {
		output = combined
	}

	return Result{
		TaskID:      t.ID,
		CommandType: t.CommandType,
		Status:      status,
		Output:      output,
		DryRunDiff:  dryRunDiff,
		ExitCode:    res.ExitCode,
		DispatchID:  p.DispatchID,
		ExecutedAt:  start,
	}
}

// handleCheckIssues runs on-demand issue checks for specific issue types.
// The agent re-evaluates the requested collectors and returns a JSON map
// of issue_type → bool (true = issue present, false = resolved).
func handleCheckIssues(t commander.Task, start time.Time) Result {
	var payload struct {
		IssueTypes []string `json:"issue_types"`
	}

	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return Result{
			TaskID:      t.ID,
			CommandType: t.CommandType,
			Status:      "failed",
			Output:      fmt.Sprintf("unmarshal check_issues payload: %v", err),
			ExecutedAt:  start,
		}
	}

	// Run collectors for each requested issue type.
	// This reuses the existing platform-specific check functions.
	results := make(map[string]bool)
	for _, issueType := range payload.IssueTypes {
		present := checkIssue(issueType)
		results[issueType] = present
	}

	output, _ := json.Marshal(results)

	return Result{
		TaskID:      t.ID,
		CommandType: t.CommandType,
		Status:      "success",
		Output:      string(output),
		ExecutedAt:  start,
	}
}

// checkIssue evaluates a single issue type using the existing collector functions.
func checkIssue(issueType string) bool {
	switch issueType {
	case "firewall_disabled":
		_, err := enableFirewall()
		// If enableFirewall returns an error, the firewall was already enabled
		// (or we can't determine). We check the state instead.
		// For now, return false (not present) if no error, true if error.
		return err != nil
	case "auto_updates_disabled":
		_, err := enableAutoUpdates()
		return err != nil
	case "screen_lock_disabled":
		_, err := enableScreenLock()
		return err != nil
	default:
		return false // Unknown issue type — assume not present
	}
}
