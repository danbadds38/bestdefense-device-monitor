package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all agent configuration. Stored at C:\ProgramData\BestDefense\config.json.
type Config struct {
	RegistrationKey    string `json:"registration_key"`
	// AgentID is assigned by the server on first successful check-in and persisted here.
	// It is sent as X-Agent-ID on all subsequent requests.
	AgentID            string `json:"agent_id,omitempty"`
	APIEndpoint        string `json:"api_endpoint"`
	CommandsEndpoint   string `json:"commands_endpoint"`
	TaskResultEndpoint string `json:"task_result_endpoint"`
	CheckIntervalHours int    `json:"check_interval_hours"`
	AgentVersion       string `json:"agent_version"`
	LogLevel           string `json:"log_level"`
	LogFile            string `json:"log_file"`
	MaxLogSizeMB       int    `json:"max_log_size_mb"`
	MaxLogBackups      int    `json:"max_log_backups"`
	HTTPTimeoutSeconds int    `json:"http_timeout_seconds"`
	RetryAttempts      int    `json:"retry_attempts"`
	RetryDelaySeconds  int    `json:"retry_delay_seconds"`
}

// DataDir returns the path to the agent's data directory.
func DataDir() string {
	return dataDir()
}

// ConfigPath returns the full path to the config file.
// The BESTDEFENSE_CONFIG_PATH environment variable overrides the default location
// (used in tests).
func ConfigPath() string {
	if p := os.Getenv("BESTDEFENSE_CONFIG_PATH"); p != "" {
		return p
	}
	return filepath.Join(dataDir(), "config.json")
}

// EnsureDataDir creates the data and logs directories if they don't exist.
func EnsureDataDir() error {
	dirs := []string{
		dataDir(),
		filepath.Join(dataDir(), "logs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

// Load reads the config file and returns a Config. Missing fields are filled from defaults.
func Load() (*Config, error) {
	cfg := Default()
	path := ConfigPath()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config %s: %w", path, err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Save writes cfg to the config file, creating it if necessary.
func Save(cfg *Config) error {
	if err := EnsureDataDir(); err != nil {
		return err
	}

	f, err := os.Create(ConfigPath())
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func (c *Config) validate() error {
	if c.RegistrationKey == "" {
		return fmt.Errorf("registration_key is required")
	}
	if c.APIEndpoint == "" {
		c.APIEndpoint = DefaultAPIEndpoint
	}
	if c.CommandsEndpoint == "" {
		c.CommandsEndpoint = DefaultCommandsEndpoint
	}
	if c.TaskResultEndpoint == "" {
		c.TaskResultEndpoint = DefaultTaskResultEndpoint
	}
	if c.CheckIntervalHours <= 0 {
		c.CheckIntervalHours = DefaultCheckIntervalHours
	}
	if c.HTTPTimeoutSeconds <= 0 {
		c.HTTPTimeoutSeconds = DefaultHTTPTimeoutSeconds
	}
	if c.RetryAttempts <= 0 {
		c.RetryAttempts = DefaultRetryAttempts
	}
	if c.RetryDelaySeconds <= 0 {
		c.RetryDelaySeconds = DefaultRetryDelaySeconds
	}
	return nil
}
