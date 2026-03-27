//go:build linux

package executor

import (
	"os/exec"
	"strings"
)

func init() {
	enableScreenLock  = linuxEnableScreenLock
	enableAutoUpdates = linuxEnableAutoUpdates
	requestReboot     = linuxRequestReboot
}

func linuxEnableScreenLock() (string, error) {
	return runSequential([][]string{
		{"gsettings", "set", "org.gnome.desktop.screensaver", "lock-enabled", "true"},
		{"gsettings", "set", "org.gnome.desktop.session", "idle-delay", "uint32 300"},
	})
}

func linuxEnableAutoUpdates() (string, error) {
	// Try Debian/Ubuntu (unattended-upgrades) first, fall back to RHEL/Fedora (dnf-automatic).
	if out, err := exec.Command("systemctl", "enable", "--now", "unattended-upgrades").CombinedOutput(); err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	out, err := exec.Command("systemctl", "enable", "--now", "dnf-automatic").CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func linuxRequestReboot() (string, error) {
	out, err := exec.Command("shutdown", "-r", "+1", "BestDefense Agent scheduled reboot").CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
