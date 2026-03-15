package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempConfig sets BESTDEFENSE_CONFIG_PATH to a temp file for the duration
// of a test, then restores the original value on cleanup.
func withTempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("BESTDEFENSE_CONFIG_PATH", path)
	return path
}

func TestDefaultHasCorrectEndpoints(t *testing.T) {
	cfg := Default()

	if cfg.APIEndpoint != DefaultAPIEndpoint {
		t.Errorf("APIEndpoint = %q, want %q", cfg.APIEndpoint, DefaultAPIEndpoint)
	}
	if cfg.CommandsEndpoint != DefaultCommandsEndpoint {
		t.Errorf("CommandsEndpoint = %q, want %q", cfg.CommandsEndpoint, DefaultCommandsEndpoint)
	}
	if cfg.TaskResultEndpoint != DefaultTaskResultEndpoint {
		t.Errorf("TaskResultEndpoint = %q, want %q", cfg.TaskResultEndpoint, DefaultTaskResultEndpoint)
	}
}

func TestDefaultAPIEndpointPointsToCheckin(t *testing.T) {
	if DefaultAPIEndpoint != "https://app.bestdefense.io/agent/checkin" {
		t.Errorf("DefaultAPIEndpoint = %q, want /agent/checkin URL", DefaultAPIEndpoint)
	}
}

func TestDefaultAgentIDIsEmpty(t *testing.T) {
	cfg := Default()
	if cfg.AgentID != "" {
		t.Errorf("AgentID should be empty on a new config, got %q", cfg.AgentID)
	}
}

func TestSaveAndLoad(t *testing.T) {
	withTempConfig(t)

	original := Default()
	original.RegistrationKey = "test-reg-key-abc123"
	original.AgentVersion = "1.2.3"

	if err := Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.RegistrationKey != original.RegistrationKey {
		t.Errorf("RegistrationKey = %q, want %q", loaded.RegistrationKey, original.RegistrationKey)
	}
	if loaded.AgentVersion != original.AgentVersion {
		t.Errorf("AgentVersion = %q, want %q", loaded.AgentVersion, original.AgentVersion)
	}
}

func TestAgentIDRoundtrip(t *testing.T) {
	withTempConfig(t)

	cfg := Default()
	cfg.RegistrationKey = "test-reg-key-roundtrip"
	cfg.AgentID = "42"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.AgentID != "42" {
		t.Errorf("AgentID after roundtrip = %q, want %q", loaded.AgentID, "42")
	}
}

func TestAgentIDOmittedFromJSONWhenEmpty(t *testing.T) {
	withTempConfig(t)

	cfg := Default()
	cfg.RegistrationKey = "test-omit-agentid"
	// AgentID intentionally left empty

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("reading config file: %v", err)
	}

	if strings.Contains(string(data), `"agent_id"`) {
		t.Error(`config JSON should not contain "agent_id" key when AgentID is empty (omitempty)`)
	}
}

func TestLoadFillsMissingEndpointsFromDefaults(t *testing.T) {
	withTempConfig(t)

	// Write a minimal config with only registration_key
	minimal := `{"registration_key":"minimal-key"}`
	if err := os.WriteFile(ConfigPath(), []byte(minimal), 0600); err != nil {
		t.Fatalf("writing minimal config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.CommandsEndpoint != DefaultCommandsEndpoint {
		t.Errorf("CommandsEndpoint not filled from defaults: got %q", loaded.CommandsEndpoint)
	}
	if loaded.TaskResultEndpoint != DefaultTaskResultEndpoint {
		t.Errorf("TaskResultEndpoint not filled from defaults: got %q", loaded.TaskResultEndpoint)
	}
}

