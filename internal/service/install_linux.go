//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const unitPath = "/etc/systemd/system/bestdefense-monitor.service"

const unitTemplate = `[Unit]
Description=BestDefense Device Monitor
Documentation=https://github.com/bestdefense/bestdefense-device-monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.ExePath}} run
Restart=always
RestartSec=30
User=root

[Install]
WantedBy=multi-user.target
`

// Install writes a systemd unit file and enables + starts the service.
// Requires root (sudo).
func Install(exePath string) error {
	abs, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Write unit file
	f, err := os.Create(unitPath)
	if err != nil {
		return fmt.Errorf("creating unit file %s (requires root): %w", unitPath, err)
	}
	defer f.Close()

	tmpl, err := template.New("unit").Parse(unitTemplate)
	if err != nil {
		return fmt.Errorf("parsing unit template: %w", err)
	}
	if err := tmpl.Execute(f, struct{ ExePath string }{ExePath: abs}); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	// Reload systemd, enable, and start
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("systemctl", "enable", "--now", ServiceName).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable --now: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// Uninstall stops, disables, and removes the systemd unit.
// Requires root (sudo).
func Uninstall() error {
	exec.Command("systemctl", "disable", "--now", ServiceName).Run()
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file %s: %w", unitPath, err)
	}
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// Status returns the running state of the systemd service.
func Status() (string, error) {
	out, err := exec.Command("systemctl", "is-active", ServiceName).Output()
	if err != nil {
		// systemctl is-active exits non-zero for inactive/unknown
		state := strings.TrimSpace(string(out))
		if state == "" {
			return "not installed", nil
		}
		return state, nil
	}
	return strings.TrimSpace(string(out)), nil
}
