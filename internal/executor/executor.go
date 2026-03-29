package executor

import (
	"fmt"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
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
}

// Run executes each task and returns one Result per task.
// kp is the agent's current identity key pair. If a rotate_keys command succeeds,
// *kp is updated in place so subsequent requests use the new private key.
// Pass nil for kp if key rotation is not supported in this context.
// Unknown command types produce a "failed" result without calling the OS.
func Run(tasks []commander.Task, kp *identity.KeyPair) []Result {
	results := make([]Result, 0, len(tasks))
	for _, t := range tasks {
		results = append(results, runTask(t, kp))
	}
	return results
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
