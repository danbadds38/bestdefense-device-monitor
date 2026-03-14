//go:build linux

package collector

import (
	"os"
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

// knownAVServices maps display names to systemd service names.
var knownAVServices = map[string]string{
	"CrowdStrike Falcon":  "falcon-sensor",
	"SentinelOne":         "sentinelone",
	"Carbon Black":        "cbagentd",
	"Malwarebytes":        "mfemms",
	"Sophos":              "sophos-spl-managementagent",
	"ESET":                "esets_daemon",
	"Trend Micro":         "ds_agent",
	"Cylance":             "cylancesvc",
	"Palo Alto Cortex XDR": "traps_pmd",
}

func collectAntivirus() (result reporter.AntivirusInfo) {
	err := safeCollect("antivirus", func() error {
		foundProduct := ""

		for displayName, svcName := range knownAVServices {
			out, err := exec.Command("systemctl", "is-active", svcName).Output()
			if err == nil && strings.TrimSpace(string(out)) == "active" {
				foundProduct = displayName
				result.RealtimeProtectionEnabled = true
				result.OnAccessProtectionEnabled = true
				break
			}
		}

		// ClamAV fallback — binary check
		if foundProduct == "" {
			if _, err := os.Stat("/usr/bin/clamscan"); err == nil {
				foundProduct = "ClamAV"
				// Check if clamd daemon is running
				if out, err := exec.Command("systemctl", "is-active", "clamav-daemon").Output(); err == nil {
					result.RealtimeProtectionEnabled = strings.TrimSpace(string(out)) == "active"
				}
			}
		}

		if foundProduct != "" {
			result.ProductStatus = "active"
			result.WindowsDefenderEnabled = false // not applicable on Linux
			result.AMServiceEnabled = true
		} else {
			result.ProductStatus = "not_detected"
		}

		// Definition version — ClamAV specific
		if out, err := exec.Command("clamscan", "--version").Output(); err == nil {
			parts := strings.Fields(string(out))
			if len(parts) >= 2 {
				result.DefinitionVersion = parts[len(parts)-1]
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
