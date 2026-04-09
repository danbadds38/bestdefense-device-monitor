package service

import (
	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
)

// InstallOptions holds configuration values passed to Install and UpdateConfig.
type InstallOptions struct {
	RegistrationKey    string
	Host               string // "" = keep existing / use default
	CheckIntervalHours int    // 0 = keep existing / use default
	LogLevel           string // "" = keep existing / use default
	Force              bool   // force reinstall even if service is already installed
}

// UpdateConfig merges opts into the persisted config.json without touching
// the OS-level service registration. Call Restart() afterwards to apply.
func UpdateConfig(opts InstallOptions) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	if opts.RegistrationKey != "" {
		cfg.RegistrationKey = opts.RegistrationKey
	}
	if opts.Host != "" {
		cfg.APIEndpoint, cfg.CommandsEndpoint, cfg.TaskResultEndpoint, cfg.RotateKeyEndpoint =
			config.EndpointsFromHost(opts.Host)
	}
	if opts.CheckIntervalHours > 0 {
		cfg.CheckIntervalHours = opts.CheckIntervalHours
	}
	if opts.LogLevel != "" {
		cfg.LogLevel = opts.LogLevel
	}
	return config.Save(cfg)
}
