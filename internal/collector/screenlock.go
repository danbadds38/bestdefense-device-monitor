//go:build windows

package collector

import (
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"golang.org/x/sys/windows/registry"
)

func collectScreenLock() (result reporter.ScreenLockInfo) {
	err := safeCollect("screenlock", func() error {
		// Check machine-wide Group Policy first (takes precedence over user settings)
		policyKey, policyErr := registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\Policies\Microsoft\Windows\Control Panel\Desktop`,
			registry.QUERY_VALUE)

		var timeout int
		var active, secure bool
		var fromPolicy bool

		if policyErr == nil {
			defer policyKey.Close()
			if v, _, err := policyKey.GetStringValue("ScreenSaveTimeOut"); err == nil {
				if t, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					timeout = t
					fromPolicy = true
				}
			}
			if v, _, err := policyKey.GetStringValue("ScreenSaveActive"); err == nil {
				active = strings.TrimSpace(v) == "1"
				if fromPolicy {
					// Only override if policy also controls active
				}
			}
			if v, _, err := policyKey.GetStringValue("ScreenSaverIsSecure"); err == nil {
				secure = strings.TrimSpace(v) == "1"
			}
		}

		if !fromPolicy {
			// Fall back to the current user's HKCU settings
			// When running as SYSTEM (service context), we read from the default user hive.
			// This is a best-effort read; the service will log a note if it can't access HKCU.
			userKey, err := registry.OpenKey(registry.CURRENT_USER,
				`Control Panel\Desktop`, registry.QUERY_VALUE)
			if err == nil {
				defer userKey.Close()
				if v, _, err := userKey.GetStringValue("ScreenSaveTimeOut"); err == nil {
					if t, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
						timeout = t
					}
				}
				if v, _, err := userKey.GetStringValue("ScreenSaveActive"); err == nil {
					active = strings.TrimSpace(v) == "1"
				}
				if v, _, err := userKey.GetStringValue("ScreenSaverIsSecure"); err == nil {
					secure = strings.TrimSpace(v) == "1"
				}
			}
		}

		result.ScreensaverTimeoutSeconds = timeout
		result.ScreensaverEnabled = active
		result.ScreensaverRequiresPassword = secure

		// Check lock on sleep via power settings registry
		// HKLM\SOFTWARE\Policies\Microsoft\Power\PowerSettings\<GUID>
		// or simpler: check if console lock display off timeout is set via GPO
		result.LockOnSleep = checkLockOnSleep()

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// checkLockOnSleep checks if the machine is configured to lock on sleep/display off.
func checkLockOnSleep() bool {
	// Check Group Policy for console lock display off timeout
	// HKLM\Software\Policies\Microsoft\Power\PowerSettings\0e796bdb-100d-47d6-a2d5-f7d2daa51f51
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`Software\Policies\Microsoft\Power\PowerSettings\0e796bdb-100d-47d6-a2d5-f7d2daa51f51`,
		registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		// If this key exists and DCSettingIndex > 0, locking is enforced
		if v, _, err := k.GetIntegerValue("DCSettingIndex"); err == nil && v > 0 {
			return true
		}
		if v, _, err := k.GetIntegerValue("ACSettingIndex"); err == nil && v > 0 {
			return true
		}
	}
	return false
}
