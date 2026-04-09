package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/collector"
	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/bestdefense/bestdefense-device-monitor/internal/service"
)

// Injected at build time via -ldflags
var (
	Version     = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
)

// isWindowsService and runAsService are defined in platform-specific files:
//
//	main_windows.go  — uses golang.org/x/sys/windows/svc
//	main_unix.go     — uses signal handling

func main() {
	if isWindowsService() {
		runAsService()
		return
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "install":
		cmdInstall(args)
	case "uninstall":
		cmdUninstall()
	case "run":
		// Called by launchd/systemd/SCM to run as a daemon
		runAsService()
	case "check":
		cmdCheck(args)
	case "status":
		cmdStatus()
	case "version":
		cmdVersion()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func cmdInstall(args []string) {
	var (
		key      string
		host     string
		interval int
		logLevel string
		force    bool
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key", "-key":
			if i+1 < len(args) {
				i++
				key = args[i]
			}
		case "--host", "-host":
			if i+1 < len(args) {
				i++
				host = args[i]
			}
		case "--interval", "-interval":
			if i+1 < len(args) {
				i++
				if n, err := strconv.Atoi(args[i]); err == nil && n > 0 {
					interval = n
				}
			}
		case "--log-level", "-log-level":
			if i+1 < len(args) {
				i++
				logLevel = args[i]
			}
		case "--force", "-force":
			force = true
		}
	}

	// Env var fallbacks (CLI flag takes precedence over env var)
	if key == "" {
		key = os.Getenv("BESTDEFENSE_KEY")
	}
	if host == "" {
		host = os.Getenv("BESTDEFENSE_HOST")
	}
	if interval == 0 {
		if v := os.Getenv("BESTDEFENSE_INTERVAL"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				interval = n
			}
		}
	}
	if logLevel == "" {
		logLevel = os.Getenv("BESTDEFENSE_LOG_LEVEL")
	}

	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key <registration_key> is required")
		fmt.Fprintln(os.Stderr, "       (or set BESTDEFENSE_KEY environment variable)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage: bestdefense-device-monitor install --key <key> [--host <url>] [--interval <hours>] [--log-level <level>] [--force]")
		os.Exit(1)
	}

	// Build config from defaults + provided options
	cfg := config.Default()
	cfg.RegistrationKey = key
	cfg.AgentVersion = Version
	if host != "" {
		cfg.APIEndpoint, cfg.CommandsEndpoint, cfg.TaskResultEndpoint, cfg.RotateKeyEndpoint =
			config.EndpointsFromHost(host)
	}
	if interval > 0 {
		cfg.CheckIntervalHours = interval
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}

	if err := config.EnsureDataDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
		os.Exit(1)
	}
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	opts := service.InstallOptions{
		RegistrationKey:    key,
		Host:               host,
		CheckIntervalHours: interval,
		LogLevel:           logLevel,
		Force:              force,
	}

	if err := service.Install(exePath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("BestDefense Device Monitor installed and started successfully.")
	fmt.Printf("Config: %s\n", config.ConfigPath())
	fmt.Printf("Host:   %s\n", hostFromEndpoint(cfg.APIEndpoint))
	fmt.Printf("Logs:   %s\n", cfg.LogFile)
}

func cmdUninstall() {
	if err := service.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to uninstall service: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("BestDefense Device Monitor service removed.")
	fmt.Printf("Data directory preserved at: %s\n", config.DataDir())
}

// cmdCheck runs a one-shot compliance check.
//
// Exit codes:
//
//	0 — success (check complete; report sent if --send)
//	1 — configuration error (missing key when --send required)
//	2 — network error (could not reach host)
//	3 — authentication error (key rejected by server)
func cmdCheck(args []string) {
	var (
		key  string
		host string
		send bool
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key", "-key":
			if i+1 < len(args) {
				i++
				key = args[i]
			}
		case "--host", "-host":
			if i+1 < len(args) {
				i++
				host = args[i]
			}
		case "--send", "-send":
			send = true
		}
	}

	// Env var fallbacks
	if key == "" {
		key = os.Getenv("BESTDEFENSE_KEY")
	}
	if host == "" {
		host = os.Getenv("BESTDEFENSE_HOST")
	}

	var cfg *config.Config
	if key != "" {
		// Build config from CLI/env — no config file required
		cfg = config.Default()
		cfg.RegistrationKey = key
		if host != "" {
			cfg.APIEndpoint, cfg.CommandsEndpoint, cfg.TaskResultEndpoint, cfg.RotateKeyEndpoint =
				config.EndpointsFromHost(host)
		}
	} else {
		// Load from file; fall back to unconfigured defaults if missing
		var err error
		cfg, err = config.Load()
		if err != nil {
			cfg = config.Default()
		}
	}
	cfg.AgentVersion = Version

	if send && cfg.RegistrationKey == "" {
		fmt.Fprintln(os.Stderr, "Error: --key <registration_key> is required when using --send")
		fmt.Fprintln(os.Stderr, "       (or set BESTDEFENSE_KEY environment variable)")
		os.Exit(1)
	}

	report := collector.Collect(cfg)
	report.AgentVersion = Version

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode report: %v\n", err)
		os.Exit(1)
	}

	if !send {
		if cfg.RegistrationKey == "" {
			fmt.Fprintln(os.Stderr, "\nNote: Run 'install --key <key>' to activate automatic reporting.")
		}
		return
	}

	r := reporter.New(cfg)
	if err := r.Send(report); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "status 401") || strings.Contains(errMsg, "status 403") {
			fmt.Fprintf(os.Stderr, "Authentication error (registration key rejected): %v\n", err)
			os.Exit(3)
		}
		fmt.Fprintf(os.Stderr, "Network error: %v\n", err)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "Report sent successfully.")
}

func cmdStatus() {
	svcStatus, err := service.Status()
	if err != nil {
		fmt.Printf("Service:        %s [not installed]\n", config.ServiceName)
	} else {
		fmt.Printf("Service:        %s [%s]\n", config.ServiceName, svcStatus)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config:         not found (%v)\n", err)
	} else {
		agentID := cfg.AgentID
		if agentID == "" {
			agentID = "not enrolled yet"
		}
		fmt.Printf("Config:         %s\n", config.ConfigPath())
		fmt.Printf("Host:           %s\n", hostFromEndpoint(cfg.APIEndpoint))
		fmt.Printf("Agent ID:       %s\n", agentID)
		fmt.Printf("Registration:   %s\n", maskKey(cfg.RegistrationKey))
		fmt.Printf("Check interval: %dh\n", cfg.CheckIntervalHours)
		fmt.Printf("Log level:      %s\n", cfg.LogLevel)
		fmt.Printf("Log file:       %s\n", cfg.LogFile)
	}

	fmt.Printf("Agent version:  %s (build %s, %s)\n", Version, BuildCommit, BuildDate)
}

func cmdVersion() {
	fmt.Printf("bestdefense-device-monitor %s\n", Version)
	fmt.Printf("Build commit: %s\n", BuildCommit)
	fmt.Printf("Build date:   %s\n", BuildDate)
}

// hostFromEndpoint extracts the base host URL from a full endpoint URL.
// e.g. "https://app.bestdefense.io/agent/checkin" → "https://app.bestdefense.io"
func hostFromEndpoint(apiEndpoint string) string {
	if i := strings.Index(apiEndpoint, "/agent/"); i > 0 {
		return apiEndpoint[:i]
	}
	return apiEndpoint
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func printUsage() {
	fmt.Println(`BestDefense Device Monitor - Security compliance agent

Usage:
  bestdefense-device-monitor <command> [options]

Commands:
  install   Install and start the monitoring service (requires elevation)
  uninstall Stop and remove the monitoring service (requires elevation)
  check     Run a one-time check and print JSON to stdout
  status    Show service status and configuration
  version   Show version information

Install options:
  --key <registration_key>   Registration key (required; or BESTDEFENSE_KEY env var)
  --host <url>               Base API URL (default: https://app.bestdefense.io;
                             or BESTDEFENSE_HOST env var)
  --interval <hours>         Check interval in hours (default: 4;
                             or BESTDEFENSE_INTERVAL env var)
  --log-level <level>        Log level: debug|info|warn|error (default: info;
                             or BESTDEFENSE_LOG_LEVEL env var)
  --force                    Reinstall even if service is already installed

Check options:
  --key <registration_key>   Use this key instead of reading from config
  --host <url>               Use this host instead of reading from config
  --send                     Send the report to the server (exit 0/2/3)

Check exit codes:
  0  Success
  1  Configuration error (missing key when --send is used)
  2  Network error (could not reach host)
  3  Authentication error (key rejected by server)

Examples:
  bestdefense-device-monitor install --key BD-xxx
  bestdefense-device-monitor install --key BD-xxx --host https://acme.bestdefense.io
  BESTDEFENSE_KEY=BD-xxx bestdefense-device-monitor install
  bestdefense-device-monitor check --key BD-xxx --host https://acme.bestdefense.io --send
  bestdefense-device-monitor status
  bestdefense-device-monitor uninstall

Source code: https://github.com/bestdefense/bestdefense-device-monitor`)
}
