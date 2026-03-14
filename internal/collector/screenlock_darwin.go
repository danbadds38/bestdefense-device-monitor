//go:build darwin

package collector

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectScreenLock() (result reporter.ScreenLockInfo) {
	err := safeCollect("screen_lock", func() error {
		// idleTime: screensaver timeout in seconds (0 = never)
		if val := defaultsRead("com.apple.screensaver", "idleTime"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				result.ScreensaverEnabled = true
				result.ScreensaverTimeoutSeconds = n
			}
		}

		// askForPassword: 1 = require password to unlock
		if val := defaultsRead("com.apple.screensaver", "askForPassword"); val == "1" {
			result.ScreensaverRequiresPassword = true
		}

		// Lock on sleep: check if "Require password ... after sleep" is enabled via
		// com.apple.screensaver askForPasswordDelay == 0
		if val := defaultsRead("com.apple.screensaver", "askForPasswordDelay"); val == "0" {
			result.LockOnSleep = result.ScreensaverRequiresPassword
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// defaultsRead reads a value from the macOS defaults system.
// Returns empty string if the key doesn't exist or the command fails.
func defaultsRead(domain, key string) string {
	out, err := exec.Command("defaults", "read", domain, key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
