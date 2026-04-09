//go:build windows

package service

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceDisplayName = "BestDefense Device Monitor"
	serviceDescription = "Collects security compliance data and reports to BestDefense. " +
		"Source code: https://github.com/bestdefense/bestdefense-device-monitor"
)

// Install registers the Windows Service, registers the Event Log source,
// adds a firewall outbound allow rule, and starts the service.
// Requires the process to be running as an Administrator.
//
// Idempotent: if the service is already installed and opts.Force is false,
// Install restarts the service to pick up any config changes. If opts.Force
// is true the existing service is removed and reinstalled cleanly.
func Install(exePath string, opts InstallOptions) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connecting to Service Control Manager (requires elevation): %w", err)
	}
	defer m.Disconnect()

	// Check if already installed
	existing, err := m.OpenService(ServiceName)
	if err == nil {
		existing.Close()
		if !opts.Force {
			// Config was already written by the caller (cmdInstall). Just restart.
			return Restart()
		}
		// Force reinstall: remove the existing service first.
		if err := Uninstall(); err != nil {
			return fmt.Errorf("removing existing service for force reinstall: %w", err)
		}
	}

	s, err := m.CreateService(ServiceName, exePath, mgr.Config{
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		StartType:        mgr.StartAutomatic,
		ServiceStartName: "LocalSystem",
	}, "run") // pass "run" arg so the service knows it's under SCM
	if err != nil {
		return fmt.Errorf("creating service: %w", err)
	}
	defer s.Close()

	// Set recovery actions: restart on failure, 3 attempts, 1-minute delay
	err = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400) // reset period: 24 hours
	if err != nil {
		// Non-fatal: log but don't fail installation
		fmt.Printf("Warning: could not set recovery actions: %v\n", err)
	}

	// Register Windows Event Log source
	if err := eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		// Non-fatal if already registered
		fmt.Printf("Warning: event log registration: %v\n", err)
	}

	// Add Windows Firewall outbound allow rule for the binary
	addFirewallRule(exePath)

	// Start the service
	if err := s.Start(); err != nil {
		return fmt.Errorf("starting service: %w", err)
	}

	return nil
}

// Restart stops and starts the Windows service.
// Requires elevation.
func Restart() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connecting to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("opening service: %w", err)
	}
	defer s.Close()

	// Stop if running
	status, err := s.Query()
	if err == nil && status.State == svc.Running {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("stopping service: %w", err)
		}
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)
			st, err := s.Query()
			if err != nil || st.State == svc.Stopped {
				break
			}
		}
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("starting service: %w", err)
	}
	return nil
}

// Uninstall stops and removes the Windows Service.
// Requires elevation.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connecting to SCM (requires elevation): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", ServiceName, err)
	}
	defer s.Close()

	// Stop if running
	status, err := s.Query()
	if err == nil && status.State == svc.Running {
		if _, err := s.Control(svc.Stop); err != nil {
			fmt.Printf("Warning: could not stop service: %v\n", err)
		}
		// Wait for stop
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			st, err := s.Query()
			if err != nil || st.State == svc.Stopped {
				break
			}
		}
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("deleting service: %w", err)
	}

	// Remove event log source registration
	eventlog.Remove(ServiceName)

	// Remove firewall rule
	removeFirewallRule()

	return nil
}

// Status returns a human-readable string describing the service state.
func Status() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("connecting to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return "", fmt.Errorf("service not installed: %w", err)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("querying service: %w", err)
	}

	switch status.State {
	case svc.Running:
		return "running", nil
	case svc.Stopped:
		return "stopped", nil
	case svc.Paused:
		return "paused", nil
	case svc.StartPending:
		return "starting", nil
	case svc.StopPending:
		return "stopping", nil
	default:
		return fmt.Sprintf("unknown (%d)", status.State), nil
	}
}
