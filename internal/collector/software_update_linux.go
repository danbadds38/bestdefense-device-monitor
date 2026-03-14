//go:build linux

package collector

import (
	"bufio"
	"os"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectSoftwareUpdate() (result reporter.SoftwareUpdateInfo) {
	err := safeCollect("software_update", func() error {
		// Debian/Ubuntu: check /etc/apt/apt.conf.d/20auto-upgrades
		if autoUpdatesEnabled, auOption := parseAptAutoUpgrades(); autoUpdatesEnabled || auOption != "" {
			result.AutomaticUpdatesEnabled = autoUpdatesEnabled
			result.AUOption = auOption
			return nil
		}

		// RHEL/Fedora: check /etc/dnf/automatic.conf
		if autoUpdatesEnabled, auOption := parseDnfAutomatic(); autoUpdatesEnabled || auOption != "" {
			result.AutomaticUpdatesEnabled = autoUpdatesEnabled
			result.AUOption = auOption
			return nil
		}

		// openSUSE: check /etc/sysconfig/automatic_update
		if data, err := os.ReadFile("/etc/sysconfig/automatic_update"); err == nil {
			if strings.Contains(string(data), "AOU_ENABLE_CRONJOB=\"true\"") {
				result.AutomaticUpdatesEnabled = true
				result.AUOption = "auto_install"
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// parseAptAutoUpgrades reads /etc/apt/apt.conf.d/20auto-upgrades
// Returns (enabled, auOption).
func parseAptAutoUpgrades() (bool, string) {
	f, err := os.Open("/etc/apt/apt.conf.d/20auto-upgrades")
	if err != nil {
		return false, ""
	}
	defer f.Close()

	var updatePackageLists, unattendedUpgrade bool
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "APT::Periodic::Update-Package-Lists") {
			updatePackageLists = strings.Contains(line, `"1"`)
		}
		if strings.HasPrefix(line, "APT::Periodic::Unattended-Upgrade") {
			unattendedUpgrade = strings.Contains(line, `"1"`)
		}
	}

	if unattendedUpgrade {
		return true, "auto_install"
	}
	if updatePackageLists {
		return true, "auto_download"
	}
	return false, "disabled"
}

// parseDnfAutomatic reads /etc/dnf/automatic.conf.
// Returns (enabled, auOption).
func parseDnfAutomatic() (bool, string) {
	f, err := os.Open("/etc/dnf/automatic.conf")
	if err != nil {
		return false, ""
	}
	defer f.Close()

	applyUpdates := ""
	downloadUpdates := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "apply_updates") {
			_, val, _ := strings.Cut(line, "=")
			applyUpdates = strings.TrimSpace(val)
		}
		if strings.HasPrefix(line, "download_updates") {
			_, val, _ := strings.Cut(line, "=")
			downloadUpdates = strings.TrimSpace(val)
		}
	}

	if applyUpdates == "yes" {
		return true, "auto_install"
	}
	if downloadUpdates == "yes" {
		return true, "auto_download"
	}
	return false, "disabled"
}
