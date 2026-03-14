//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const plistPath = "/Library/LaunchDaemons/io.bestdefense.monitor.plist"

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.bestdefense.monitor</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{.ExePath}}</string>
    <string>run</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>UserName</key>
  <string>root</string>
  <key>StandardOutPath</key>
  <string>/Library/Application Support/BestDefense/logs/stdout.log</string>
  <key>StandardErrorPath</key>
  <string>/Library/Application Support/BestDefense/logs/stderr.log</string>
</dict>
</plist>
`

// Install writes a launchd plist and loads it.
// Requires root (sudo).
func Install(exePath string) error {
	// Resolve absolute path
	abs, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Write plist
	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("creating plist %s (requires root): %w", plistPath, err)
	}
	defer f.Close()

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("parsing plist template: %w", err)
	}
	if err := tmpl.Execute(f, struct{ ExePath string }{ExePath: abs}); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	// Set permissions (launchd requires root:wheel 644)
	if err := os.Chmod(plistPath, 0644); err != nil {
		return fmt.Errorf("setting plist permissions: %w", err)
	}

	// Load the daemon
	if out, err := exec.Command("launchctl", "load", "-w", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// Uninstall unloads and removes the launchd plist.
// Requires root (sudo).
func Uninstall() error {
	// Unload first (ignore error if not loaded)
	exec.Command("launchctl", "unload", "-w", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist %s: %w", plistPath, err)
	}
	return nil
}

// Status returns the running state of the launchd daemon.
func Status() (string, error) {
	out, err := exec.Command("launchctl", "list", ServiceName).CombinedOutput()
	if err != nil {
		return "not installed", nil
	}
	output := string(out)
	if strings.Contains(output, `"PID"`) || strings.Contains(output, "\tPID\t") {
		return "running", nil
	}
	return "stopped", nil
}
