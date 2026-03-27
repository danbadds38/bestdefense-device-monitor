//go:build darwin

package executor

import (
	"os/exec"
	"strings"
)

func init() {
	enableScreenLock  = darwinEnableScreenLock
	enableAutoUpdates = darwinEnableAutoUpdates
	requestReboot     = darwinRequestReboot
}

func darwinEnableScreenLock() (string, error) {
	return runSequential([][]string{
		{"defaults", "write", "com.apple.screensaver", "askForPassword", "-int", "1"},
		{"defaults", "write", "com.apple.screensaver", "askForPasswordDelay", "-int", "0"},
	})
}

func darwinEnableAutoUpdates() (string, error) {
	const plist = "/Library/Preferences/com.apple.SoftwareUpdate"
	return runSequential([][]string{
		{"defaults", "write", plist, "AutomaticCheckEnabled", "-bool", "true"},
		{"defaults", "write", plist, "AutomaticDownload", "-bool", "true"},
		{"defaults", "write", plist, "AutomaticallyInstallMacOSUpdates", "-bool", "true"},
	})
}

func darwinRequestReboot() (string, error) {
	out, err := exec.Command("shutdown", "-r", "+1").CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
