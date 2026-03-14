package config

import "path/filepath"

const (
	ServiceName    = "BestDefenseMonitor"
	ServiceDisplay = "BestDefense Device Monitor"
	ServiceDesc    = "Collects security compliance data and reports to BestDefense. See https://github.com/bestdefense/bestdefense-device-monitor for source code."

	DefaultAPIEndpoint        = "https://app.bestdefense.io/monitoring/employee/update"
	DefaultCheckIntervalHours = 4
	DefaultHTTPTimeoutSeconds = 30
	DefaultRetryAttempts      = 3
	DefaultRetryDelaySeconds  = 60
	DefaultMaxLogSizeMB       = 10
	DefaultMaxLogBackups      = 3
	DefaultLogLevel           = "info"
)

func dataDir() string {
	return `C:\ProgramData\BestDefense`
}

// Default returns a Config populated with all default values.
func Default() *Config {
	return &Config{
		RegistrationKey:    "",
		APIEndpoint:        DefaultAPIEndpoint,
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
