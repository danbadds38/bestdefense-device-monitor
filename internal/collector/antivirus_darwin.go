//go:build darwin

package collector

import (
	"os"
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

// Known third-party AV/EDR application paths on macOS
var knownAVApps = []struct {
	name string
	path string
}{
	{"CrowdStrike Falcon", "/Applications/Falcon.app"},
	{"SentinelOne", "/Applications/SentinelOne/SentinelOne.app"},
	{"Carbon Black", "/Applications/VMware Carbon Black Cloud/VMware Carbon Black Cloud.app"},
	{"Malwarebytes", "/Applications/Malwarebytes.app"},
	{"Sophos", "/Applications/Sophos/Sophos Anti-Virus.app"},
	{"Jamf Protect", "/Applications/JamfProtect.app"},
	{"Microsoft Defender", "/Applications/Microsoft Defender.app"},
	{"Bitdefender", "/Applications/Bitdefender/Bitdefender Antivirus for Mac.app"},
}

func collectAntivirus() (result reporter.AntivirusInfo) {
	err := safeCollect("antivirus", func() error {
		// Check XProtect version (built-in macOS AV)
		xprotectPlist := "/System/Library/CoreServices/XProtect.bundle/Contents/Resources/XProtect.meta.plist"
		if _, err := os.Stat(xprotectPlist); err == nil {
			// XProtect is present — read its version via PlistBuddy
			if out, err := exec.Command("/usr/libexec/PlistBuddy",
				"-c", "Print :Version", xprotectPlist).Output(); err == nil {
				result.DefinitionVersion = strings.TrimSpace(string(out))
			}
			result.WindowsDefenderEnabled = false // N/A on macOS
			result.RealtimeProtectionEnabled = true // XProtect is always active
			result.AMServiceEnabled = true
			result.ProductStatus = "xprotect_active"
		}

		// Scan for known third-party AV installations
		var found []string
		for _, av := range knownAVApps {
			if _, err := os.Stat(av.path); err == nil {
				found = append(found, av.name)
			}
		}
		if len(found) > 0 {
			result.ProductStatus = strings.Join(found, ", ")
			result.RealtimeProtectionEnabled = true
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
