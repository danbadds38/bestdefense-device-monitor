package config

import "path/filepath"

const (
	ServiceName    = "BestDefenseMonitor"
	ServiceDisplay = "BestDefense Device Monitor"
	ServiceDesc    = "Collects security compliance data and reports to BestDefense. See https://github.com/bestdefense/bestdefense-device-monitor for source code."

	DefaultAPIEndpoint         = "https://app.bestdefense.io/agent/checkin"
	DefaultCommandsEndpoint    = "https://app.bestdefense.io/agent/commands"
	DefaultTaskResultEndpoint  = "https://app.bestdefense.io/agent/task-result"
	DefaultRotateKeyEndpoint   = "https://app.bestdefense.io/agent/rotate-key"
	DefaultCheckIntervalHours  = 4
	DefaultHTTPTimeoutSeconds  = 30
	DefaultRetryAttempts       = 3
	DefaultRetryDelaySeconds   = 60
	DefaultMaxLogSizeMB        = 10
	DefaultMaxLogBackups       = 3
	DefaultLogLevel            = "info"
)

// dataDir() is defined in platform-specific files:
//   defaults_windows.go  →  C:\ProgramData\BestDefense
//   defaults_darwin.go   →  /Library/Application Support/BestDefense
//   defaults_linux.go    →  /var/lib/bestdefense

// Default returns a Config populated with all default values.
func Default() *Config {
	return &Config{
		RegistrationKey:    "",
		AgentID:            "",
		APIEndpoint:        DefaultAPIEndpoint,
		CommandsEndpoint:   DefaultCommandsEndpoint,
		TaskResultEndpoint: DefaultTaskResultEndpoint,
		RotateKeyEndpoint:  DefaultRotateKeyEndpoint,
		CheckIntervalHours: DefaultCheckIntervalHours,
		AgentVersion:       "dev",
		LogLevel:           DefaultLogLevel,
		LogFile:            filepath.Join(dataDir(), "logs", "agent.log"),
		MaxLogSizeMB:       DefaultMaxLogSizeMB,
		MaxLogBackups:      DefaultMaxLogBackups,
		HTTPTimeoutSeconds: DefaultHTTPTimeoutSeconds,
		RetryAttempts:      DefaultRetryAttempts,
		RetryDelaySeconds:  DefaultRetryDelaySeconds,
	}
}
