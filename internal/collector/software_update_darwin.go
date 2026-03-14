//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

const swUpdateDomain = "/Library/Preferences/com.apple.SoftwareUpdate"

func collectSoftwareUpdate() (result reporter.SoftwareUpdateInfo) {
	err := safeCollect("software_update", func() error {
		autoCheck := defaultsReadGlobal(swUpdateDomain, "AutomaticCheckEnabled")
		autoDownload := defaultsReadGlobal(swUpdateDomain, "AutomaticDownload")
		autoInstall := defaultsReadGlobal(swUpdateDomain, "AutomaticallyInstallMacOSUpdates")
		criticalUpdate := defaultsReadGlobal(swUpdateDomain, "CriticalUpdateInstall")

		switch {
		case autoInstall == "1" || criticalUpdate == "1":
			result.AutomaticUpdatesEnabled = true
			result.AUOption = "auto_install"
		case autoDownload == "1":
			result.AutomaticUpdatesEnabled = true
			result.AUOption = "auto_download"
		case autoCheck == "1":
			result.AutomaticUpdatesEnabled = true
			result.AUOption = "auto_check_only"
		default:
			result.AutomaticUpdatesEnabled = false
			result.AUOption = "disabled"
		}

		// Check for pending updates (softwareupdate --list exits 0 even if there are updates)
		if out, err := exec.Command("softwareupdate", "--list").Output(); err == nil {
			output := string(out)
			// If there are recommended updates, they appear as lines with "*"
			if strings.Contains(output, "* ") || strings.Contains(output, "Label:") {
				result.PendingReboot = true
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// defaultsReadGlobal reads from a plist path (not the per-user defaults domain).
func defaultsReadGlobal(domain, key string) string {
	out, err := exec.Command("defaults", "read", domain, key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
