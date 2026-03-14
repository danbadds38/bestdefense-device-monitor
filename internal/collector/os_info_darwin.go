//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectOSInfo() (result reporter.OSInfo) {
	err := safeCollect("os_info", func() error {
		result.Name = swVers("productName")
		result.Version = swVers("productVersion")
		result.BuildNumber = swVers("buildVersion")
		result.DisplayVersion = result.Version

		if out, err := exec.Command("uname", "-m").Output(); err == nil {
			arch := strings.TrimSpace(string(out))
			// Normalise to match Windows naming convention
			switch arch {
			case "arm64":
				result.Architecture = "ARM64"
			case "x86_64":
				result.Architecture = "x86_64"
			default:
				result.Architecture = arch
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

func swVers(flag string) string {
	out, err := exec.Command("sw_vers", "-"+flag).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
