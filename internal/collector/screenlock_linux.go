//go:build linux

package collector

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectScreenLock() (result reporter.ScreenLockInfo) {
	err := safeCollect("screen_lock", func() error {
		// Try GNOME gsettings first
		if timeout, err := gsettingsGet("org.gnome.desktop.screensaver", "idle-delay"); err == nil {
			if n, err := strconv.Atoi(timeout); err == nil && n > 0 {
				result.ScreensaverEnabled = true
				result.ScreensaverTimeoutSeconds = n
			}
		}

		if lockEnabled, err := gsettingsGet("org.gnome.desktop.screensaver", "lock-enabled"); err == nil {
			result.ScreensaverRequiresPassword = strings.TrimSpace(lockEnabled) == "true"
		}

		if lockDelay, err := gsettingsGet("org.gnome.desktop.screensaver", "lock-delay"); err == nil {
			if n, err := strconv.Atoi(lockDelay); err == nil {
				result.LockOnSleep = n == 0 // lock immediately on sleep
			}
		}

		// Try KDE if GNOME not found
		if !result.ScreensaverEnabled {
			if out, err := exec.Command("kreadconfig5",
				"--file", "kscreenlockerrc",
				"--group", "Daemon",
				"--key", "Timeout").Output(); err == nil {
				if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && n > 0 {
					result.ScreensaverEnabled = true
					result.ScreensaverTimeoutSeconds = n * 60 // KDE stores in minutes
				}
			}
		}

		// X11 fallback via xset
		if !result.ScreensaverEnabled {
			if out, err := exec.Command("xset", "q").Output(); err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "timeout:") {
						fields := strings.Fields(line)
						if len(fields) >= 2 {
							if n, err := strconv.Atoi(fields[1]); err == nil && n > 0 {
								result.ScreensaverEnabled = true
								result.ScreensaverTimeoutSeconds = n
							}
						}
					}
				}
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

func gsettingsGet(schema, key string) (string, error) {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(string(out))
	// gsettings wraps strings in single quotes
	val = strings.Trim(val, "'")
	return val, nil
}
