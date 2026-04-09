package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	RotateKeyEndpoint  string `json:"rotate_key_endpoint"`
	CheckIntervalHours int    `json:"check_interval_hours"`
	AgentVersion       string `json:"agent_version"`
	LogLevel           string `json:"log_level"`
	LogFile            string `json:"log_file"`
	MaxLogSizeMB       int    `json:"max_log_size_mb"`
	MaxLogBackups      int    `json:"max_log_backups"`
	HTTPTimeoutSeconds int    `json:"http_timeout_seconds"`
	RetryAttempts      int    `json:"retry_attempts"`
	RetryDelaySeconds  int    `json:"retry_delay_seconds"`
	// PublicKeyBase64 is the base64-encoded Ed25519 public key for this device.
	// It is populated at startup from the identity key file, not from config.json.
	PublicKeyBase64 string `json:"-"`
}

// EndpointsFromHost derives the four agent endpoint URLs from a base host URL.
// host should be scheme + hostname, e.g. "https://app.bestdefense.io".
// A trailing slash on host is stripped before appending paths.
func EndpointsFromHost(host string) (api, commands, taskResult, rotateKey string) {
	base := strings.TrimRight(host, "/")
	return base + "/agent/checkin",
		base + "/agent/commands",
		base + "/agent/task-result",
		base + "/agent/rotate-key"
}

// LoadFromEnv overlays environment variable values onto c.
// Only non-empty env vars are applied; existing values are preserved.
// Precedence: env var > value already set in c.
//
// Variables recognised:
//
//	BESTDEFENSE_KEY          registration key
//	BESTDEFENSE_HOST         base URL — derives all four endpoints
//	BESTDEFENSE_INTERVAL     check interval in hours (integer)
//	BESTDEFENSE_LOG_LEVEL    log level: debug|info|warn|error
func (c *Config) LoadFromEnv() {
	if v := os.Getenv("BESTDEFENSE_KEY"); v != "" {
		c.RegistrationKey = v
	}
	if v := os.Getenv("BESTDEFENSE_HOST"); v != "" {
		c.APIEndpoint, c.CommandsEndpoint, c.TaskResultEndpoint, c.RotateKeyEndpoint =
			EndpointsFromHost(v)
	}
	if v := os.Getenv("BESTDEFENSE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.CheckIntervalHours = n
		}
	}
	if v := os.Getenv("BESTDEFENSE_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
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

	cfg.LoadFromEnv() // env vars take precedence over file values

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
	if c.RotateKeyEndpoint == "" {
		c.RotateKeyEndpoint = DefaultRotateKeyEndpoint
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
