//go:build windows

package collector

import (
	"strings"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"golang.org/x/sys/windows/registry"
)

// AUOptions registry values:
// 1 = Keep my computer up to date (disabled)
// 2 = Notify for download and install
// 3 = Auto download and notify for install
// 4 = Auto download and schedule the install
// 5 = Automatic Updates is required and users can configure it
var auOptionLabels = map[uint64]string{
	1: "disabled",
	2: "notify_download",
	3: "auto_download_notify_install",
	4: "auto_install",
	5: "managed",
}

func collectSoftwareUpdate() (result reporter.SoftwareUpdateInfo) {
	err := safeCollect("software_update", func() error {
		// GPO path takes precedence
		paths := []struct {
			root registry.Key
			path string
		}{
			{registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`},
			{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update`},
		}

		for _, p := range paths {
			k, err := registry.OpenKey(p.root, p.path, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			defer k.Close()

			if v, _, err := k.GetIntegerValue("AUOptions"); err == nil {
				label, ok := auOptionLabels[v]
				if !ok {
					label = "unknown"
				}
				result.AUOption = label
				result.AutomaticUpdatesEnabled = v >= 3 // 3 or 4 means auto behavior
			}

			if v, _, err := k.GetIntegerValue("NoAutoUpdate"); err == nil {
				if v == 1 {
					result.AutomaticUpdatesEnabled = false
					result.AUOption = "disabled"
				}
			}

			if result.AUOption != "" {
				break
			}
		}

		// WSUS server
		wsusKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate`,
			registry.QUERY_VALUE)
		if err == nil {
			defer wsusKey.Close()
			if v, _, err := wsusKey.GetStringValue("WUServer"); err == nil && v != "" {
				result.WSUSServer = strPtr(strings.TrimSpace(v))
			}
		}

		// Last successful update time
		updateKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\Results\Install`,
			registry.QUERY_VALUE)
		if err == nil {
			defer updateKey.Close()
			if v, _, err := updateKey.GetStringValue("LastSuccessTime"); err == nil {
				// Format: "YYYY-MM-DD HH:MM:SS"
				t, parseErr := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(v))
				if parseErr == nil {
					t = t.UTC()
					result.LastSuccessfulUpdateTime = &t
				}
			}
		}

		// Pending reboot
		rebootKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`,
			registry.QUERY_VALUE)
		if err == nil {
			rebootKey.Close()
			result.PendingReboot = true
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
