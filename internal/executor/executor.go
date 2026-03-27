package executor

import (
	"fmt"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
)

// enableFirewall is set by the platform-specific init() in firewall_{os}.go.
// Declared as a var so tests can replace it without executing real OS commands.
var enableFirewall func() (string, error)

// Result holds the outcome of executing a single remediation task.
type Result struct {
	TaskID      int
	CommandType string
	Status      string // "success" or "failed"
	Output      string
	ExecutedAt  time.Time
}

// Run executes each task and returns one Result per task.
// Unknown command types produce a "failed" result without calling the OS.
func Run(tasks []commander.Task) []Result {
	results := make([]Result, 0, len(tasks))
	for _, t := range tasks {
		results = append(results, runTask(t))
	}
	return results
}

func runTask(t commander.Task) Result {
	start := time.Now().UTC()

	var output string
	var err error

	switch t.CommandType {
	case "enable_firewall":
		output, err = enableFirewall()
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
