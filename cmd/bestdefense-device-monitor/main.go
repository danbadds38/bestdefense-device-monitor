package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bestdefense/bestdefense-device-monitor/internal/collector"
	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/bestdefense/bestdefense-device-monitor/internal/service"
	"golang.org/x/sys/windows/svc"
)

// Injected at build time via -ldflags
var (
	Version     = "dev"
	BuildCommit = "unknown"
	BuildDate   = "unknown"
)

func main() {
	// Detect if we're running as a Windows Service (no console args from SCM)
	isService, err := svc.IsWindowsService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to detect service context: %v\n", err)
		os.Exit(1)
	}

	if isService {
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
		// Explicit run (used during development / debugging outside SCM)
		runAsService()
	case "check":
		cmdCheck()
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

func runAsService() {
	log := logging.NewEventLogger()
	if err := svc.Run(service.ServiceName, service.New(log)); err != nil {
		log.Error(fmt.Sprintf("Service failed: %v", err))
		os.Exit(1)
	}
}

func cmdInstall(args []string) {
	var key string
	for i, a := range args {
		if (a == "--key" || a == "-key") && i+1 < len(args) {
			key = args[i+1]
		}
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key <registration_key> is required")
		fmt.Fprintln(os.Stderr, "Usage: bestdefense-device-monitor.exe install --key <your_registration_key>")
		os.Exit(1)
	}

	cfg := config.Default()
	cfg.RegistrationKey = key
	cfg.AgentVersion = Version

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

	if err := service.Install(exePath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to install service: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("BestDefense Device Monitor installed and started successfully.")
	fmt.Printf("Config: %s\n", config.ConfigPath())
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

func cmdCheck() {
	cfg, err := config.Load()
	if err != nil {
		// Allow check without config (useful for IT audit before install)
		cfg = config.Default()
		cfg.RegistrationKey = "NOT_CONFIGURED"
	}

	report := collector.Collect(cfg)
	report.AgentVersion = Version

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode report: %v\n", err)
		os.Exit(1)
	}

	if cfg.RegistrationKey != "NOT_CONFIGURED" {
		fmt.Fprintln(os.Stderr, "\nNote: Run 'install --key <key>' to activate automatic reporting.")
		return
	}

	// If a key is configured, ask if they want to send
	if len(os.Args) > 2 && os.Args[2] == "--send" {
		r := reporter.New(cfg)
		if err := r.Send(report); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send report: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Report sent successfully.")
	}
}

func cmdStatus() {
	svcStatus, err := service.Status()
	if err != nil {
		fmt.Printf("Service status: not installed (%v)\n", err)
	} else {
		fmt.Printf("Service status: %s\n", svcStatus)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config: not found (%v)\n", err)
	} else {
		fmt.Printf("Config:         %s\n", config.ConfigPath())
		fmt.Printf("Registration:   %s\n", maskKey(cfg.RegistrationKey))
		fmt.Printf("API endpoint:   %s\n", cfg.APIEndpoint)
		fmt.Printf("Check interval: %dh\n", cfg.CheckIntervalHours)
	}

	fmt.Printf("Agent version:  %s (build %s, %s)\n", Version, BuildCommit, BuildDate)
}

func cmdVersion() {
	fmt.Printf("bestdefense-device-monitor %s\n", Version)
	fmt.Printf("Build commit: %s\n", BuildCommit)
	fmt.Printf("Build date:   %s\n", BuildDate)
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
  bestdefense-device-monitor.exe <command> [options]

Commands:
  install --key <registration_key>   Install and start the monitoring service (requires elevation)
  uninstall                          Stop and remove the monitoring service (requires elevation)
  check [--send]                     Run a one-time check and print JSON to stdout
  status                             Show service status and configuration
  version                            Show version information

Examples:
  bestdefense-device-monitor.exe install --key cust_abc123xyz
  bestdefense-device-monitor.exe check
  bestdefense-device-monitor.exe status
  bestdefense-device-monitor.exe uninstall

The 'check' command lets IT teams audit exactly what data is collected before deployment.
Source code: https://github.com/bestdefense/bestdefense-device-monitor`)
}
