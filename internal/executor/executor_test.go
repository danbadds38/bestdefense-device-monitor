package executor

import (
	"errors"
	"testing"

	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
)

func withFirewall(fn func() (string, error), test func()) {
	orig := enableFirewall
	enableFirewall = fn
	defer func() { enableFirewall = orig }()
	test()
}

func withScreenLock(fn func() (string, error), test func()) {
	orig := enableScreenLock
	enableScreenLock = fn
	defer func() { enableScreenLock = orig }()
	test()
}

func withAutoUpdates(fn func() (string, error), test func()) {
	orig := enableAutoUpdates
	enableAutoUpdates = fn
	defer func() { enableAutoUpdates = orig }()
	test()
}

func withReboot(fn func() (string, error), test func()) {
	orig := requestReboot
	requestReboot = fn
	defer func() { requestReboot = orig }()
	test()
}

func TestRunReturnsOneResultPerTask(t *testing.T) {
	withFirewall(func() (string, error) { return "ok", nil }, func() {
		tasks := []commander.Task{
			{ID: 1, CommandType: "enable_firewall"},
			{ID: 2, CommandType: "enable_firewall"},
		}
		results := Run(tasks)
		if len(results) != 2 {
			t.Errorf("Run() returned %d results, want 2", len(results))
		}
	})
}

func TestRunSuccessStatus(t *testing.T) {
	withFirewall(func() (string, error) { return "Firewall enabled.", nil }, func() {
		tasks := []commander.Task{{ID: 5, CommandType: "enable_firewall"}}
		results := Run(tasks)
		if results[0].Status != "success" {
			t.Errorf("Status = %q, want %q", results[0].Status, "success")
		}
		if results[0].Output != "Firewall enabled." {
			t.Errorf("Output = %q, want %q", results[0].Output, "Firewall enabled.")
		}
		if results[0].TaskID != 5 {
			t.Errorf("TaskID = %d, want 5", results[0].TaskID)
		}
	})
}

func TestRunFailedStatus(t *testing.T) {
	withFirewall(func() (string, error) { return "", errors.New("ufw: command not found") }, func() {
		tasks := []commander.Task{{ID: 9, CommandType: "enable_firewall"}}
		results := Run(tasks)
		if results[0].Status != "failed" {
			t.Errorf("Status = %q, want %q", results[0].Status, "failed")
		}
		if results[0].Output != "ufw: command not found" {
			t.Errorf("Output = %q, want error message", results[0].Output)
		}
	})
}

func TestRunUnknownCommandTypeReturnsFailed(t *testing.T) {
	tasks := []commander.Task{{ID: 3, CommandType: "self_destruct"}}
	results := Run(tasks)
	if len(results) != 1 {
		t.Fatalf("Run() returned %d results, want 1", len(results))
	}
	if results[0].Status != "failed" {
		t.Errorf("Status = %q, want %q", results[0].Status, "failed")
	}
	if results[0].TaskID != 3 {
		t.Errorf("TaskID = %d, want 3", results[0].TaskID)
	}
}

func TestRunSetsExecutedAt(t *testing.T) {
	withFirewall(func() (string, error) { return "ok", nil }, func() {
		tasks := []commander.Task{{ID: 1, CommandType: "enable_firewall"}}
		results := Run(tasks)
		if results[0].ExecutedAt.IsZero() {
			t.Error("ExecutedAt should not be zero")
		}
	})
}

func TestRunEmptyTasksReturnsEmptySlice(t *testing.T) {
	results := Run([]commander.Task{})
	if len(results) != 0 {
		t.Errorf("Run() with no tasks returned %d results, want 0", len(results))
	}
}

func TestRunEnableScreenLock(t *testing.T) {
	withScreenLock(func() (string, error) { return "Screen lock enabled.", nil }, func() {
		tasks := []commander.Task{{ID: 10, CommandType: "enable_screen_lock"}}
		results := Run(tasks)
		if results[0].Status != "success" {
			t.Errorf("Status = %q, want %q", results[0].Status, "success")
		}
		if results[0].Output != "Screen lock enabled." {
			t.Errorf("Output = %q, want %q", results[0].Output, "Screen lock enabled.")
		}
		if results[0].TaskID != 10 {
			t.Errorf("TaskID = %d, want 10", results[0].TaskID)
		}
	})
}

func TestRunEnableAutoUpdates(t *testing.T) {
	withAutoUpdates(func() (string, error) { return "Auto updates enabled.", nil }, func() {
		tasks := []commander.Task{{ID: 11, CommandType: "enable_auto_updates"}}
		results := Run(tasks)
		if results[0].Status != "success" {
			t.Errorf("Status = %q, want %q", results[0].Status, "success")
		}
		if results[0].Output != "Auto updates enabled." {
			t.Errorf("Output = %q, want %q", results[0].Output, "Auto updates enabled.")
		}
		if results[0].TaskID != 11 {
			t.Errorf("TaskID = %d, want 11", results[0].TaskID)
		}
	})
}

func TestRunRequestReboot(t *testing.T) {
	withReboot(func() (string, error) { return "Reboot scheduled.", nil }, func() {
		tasks := []commander.Task{{ID: 12, CommandType: "request_reboot"}}
		results := Run(tasks)
		if results[0].Status != "success" {
			t.Errorf("Status = %q, want %q", results[0].Status, "success")
		}
		if results[0].Output != "Reboot scheduled." {
			t.Errorf("Output = %q, want %q", results[0].Output, "Reboot scheduled.")
		}
		if results[0].TaskID != 12 {
			t.Errorf("TaskID = %d, want 12", results[0].TaskID)
		}
	})
}
