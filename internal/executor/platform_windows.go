//go:build windows

package executor

import (
	"os/exec"
	"strings"
)

func init() {
	enableScreenLock  = windowsEnableScreenLock
	enableAutoUpdates = windowsEnableAutoUpdates
	requestReboot     = windowsRequestReboot
}

func windowsEnableScreenLock() (string, error) {
	const key = `HKCU\Control Panel\Desktop`
	return runSequential([][]string{
		{"reg", "add", key, "/v", "ScreenSaveActive", "/t", "REG_SZ", "/d", "1", "/f"},
		{"reg", "add", key, "/v", "ScreenSaverIsSecure", "/t", "REG_SZ", "/d", "1", "/f"},
		{"reg", "add", key, "/v", "ScreenSaveTimeOut", "/t", "REG_SZ", "/d", "300", "/f"},
	})
}

func windowsEnableAutoUpdates() (string, error) {
	const key = `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update`
	out, err := exec.Command(
		"reg", "add", key, "/v", "AUOptions", "/t", "REG_DWORD", "/d", "4", "/f",
	).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func windowsRequestReboot() (string, error) {
	out, err := exec.Command(
		"shutdown", "/r", "/t", "60", "/c", "BestDefense Agent scheduled reboot",
	).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
