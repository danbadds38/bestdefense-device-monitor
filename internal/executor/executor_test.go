package executor

import (
	"bytes"
	"errors"
	"testing"

	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
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
		results := Run(tasks, nil)
		if len(results) != 2 {
			t.Errorf("Run() returned %d results, want 2", len(results))
		}
	})
}

func TestRunSuccessStatus(t *testing.T) {
	withFirewall(func() (string, error) { return "Firewall enabled.", nil }, func() {
		tasks := []commander.Task{{ID: 5, CommandType: "enable_firewall"}}
		results := Run(tasks, nil)
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
		results := Run(tasks, nil)
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
	results := Run(tasks, nil)
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
		results := Run(tasks, nil)
		if results[0].ExecutedAt.IsZero() {
			t.Error("ExecutedAt should not be zero")
		}
	})
}

func TestRunEmptyTasksReturnsEmptySlice(t *testing.T) {
	results := Run([]commander.Task{}, nil)
	if len(results) != 0 {
		t.Errorf("Run() with no tasks returned %d results, want 0", len(results))
	}
}

func TestRunEnableScreenLock(t *testing.T) {
	withScreenLock(func() (string, error) { return "Screen lock enabled.", nil }, func() {
		tasks := []commander.Task{{ID: 10, CommandType: "enable_screen_lock"}}
		results := Run(tasks, nil)
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
		results := Run(tasks, nil)
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
		results := Run(tasks, nil)
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

func TestRunRotateKeysWithNoFuncConfigured(t *testing.T) {
	orig := rotateKeys
	rotateKeys = nil
	defer func() { rotateKeys = orig }()

	tasks := []commander.Task{{ID: 20, CommandType: "rotate_keys"}}
	results := Run(tasks, nil)
	if len(results) != 1 {
		t.Fatalf("Run() returned %d results, want 1", len(results))
	}
	if results[0].Status != "failed" {
		t.Errorf("Status = %q, want %q", results[0].Status, "failed")
	}
	if results[0].TaskID != 20 {
		t.Errorf("TaskID = %d, want 20", results[0].TaskID)
	}
}

func TestRunRotateKeysSuccess(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/agent_identity.key")

	oldKP, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}

	newKP, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate() new: %v", err)
	}

	SetRotateKeysFunc(func(kp *identity.KeyPair) (*identity.KeyPair, error) {
		return newKP, nil
	})
	defer SetRotateKeysFunc(nil)

	tasks := []commander.Task{{ID: 21, CommandType: "rotate_keys"}}
	results := Run(tasks, oldKP)
	if results[0].Status != "success" {
		t.Errorf("Status = %q, want %q", results[0].Status, "success")
	}

	// kp should be updated in place to the new key pair
	if !oldKP.PublicKey.Equal(newKP.PublicKey) {
		t.Error("Run() did not update kp in place after rotate_keys success")
	}
}

func TestRunRotateKeysFailure(t *testing.T) {
	t.Setenv("BESTDEFENSE_IDENTITY_PATH", t.TempDir()+"/agent_identity.key")

	kp, err := identity.Generate()
	if err != nil {
		t.Fatalf("identity.Generate(): %v", err)
	}
	originalPublicKey := make([]byte, len(kp.PublicKey))
	copy(originalPublicKey, kp.PublicKey)

	SetRotateKeysFunc(func(kp *identity.KeyPair) (*identity.KeyPair, error) {
		return nil, errors.New("server rejected rotation")
	})
	defer SetRotateKeysFunc(nil)

	tasks := []commander.Task{{ID: 22, CommandType: "rotate_keys"}}
	results := Run(tasks, kp)
	if results[0].Status != "failed" {
		t.Errorf("Status = %q, want %q", results[0].Status, "failed")
	}

	// kp should be unchanged on failure
	if !bytes.Equal(kp.PublicKey, originalPublicKey) {
		t.Error("Run() modified kp on rotate_keys failure; original should be preserved")
	}
}
