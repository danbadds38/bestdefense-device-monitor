//go:build windows

package collector

import (
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"golang.org/x/sys/windows/registry"
)

// uninstallPaths are the four registry hives where installed applications are recorded.
// NOTE: We deliberately do NOT use Win32_Product (WMI) as it triggers MSI reconfigure sequences.
var uninstallPaths = []struct {
	root   registry.Key
	path   string
	source string
}{
	{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "HKLM_x64"},
	{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, "HKLM_x86"},
	{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "HKCU_x64"},
	{registry.CURRENT_USER, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, "HKCU_x86"},
}

func collectApplications() (result reporter.InstalledAppsInfo) {
	err := safeCollect("applications", func() error {
		seen := make(map[string]bool) // deduplicate by name+version

		for _, up := range uninstallPaths {
			k, err := registry.OpenKey(up.root, up.path, registry.ENUMERATE_SUB_KEYS)
			if err != nil {
				continue
			}

			subkeys, err := k.ReadSubKeyNames(-1)
			k.Close()
			if err != nil {
				continue
			}

			for _, subkey := range subkeys {
				fullPath := up.path + `\` + subkey
				sk, err := registry.OpenKey(up.root, fullPath, registry.QUERY_VALUE)
				if err != nil {
					continue
				}

				var app reporter.InstalledApp
				app.Source = up.source

				name, _, _ := sk.GetStringValue("DisplayName")
				app.Name = strings.TrimSpace(name)
				if app.Name == "" {
					sk.Close()
					continue
				}

				// Skip system components and updates to reduce noise
				if isSystemComponent(sk) {
					sk.Close()
					continue
				}

				ver, _, _ := sk.GetStringValue("DisplayVersion")
				app.Version = strings.TrimSpace(ver)

				pub, _, _ := sk.GetStringValue("Publisher")
				app.Publisher = strings.TrimSpace(pub)

				installDate, _, _ := sk.GetStringValue("InstallDate")
				app.InstallDate = strings.TrimSpace(installDate)

				installLoc, _, _ := sk.GetStringValue("InstallLocation")
				app.InstallLocation = strings.TrimSpace(installLoc)

				sk.Close()

				// Deduplicate
				key := app.Name + "|" + app.Version
				if seen[key] {
					continue
				}
				seen[key] = true

				result.Applications = append(result.Applications, app)
			}
		}

		result.TotalCount = len(result.Applications)
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// isSystemComponent returns true for Windows system components and updates
// that clutter the application list.
func isSystemComponent(k registry.Key) bool {
	// SystemComponent = 1 means hidden from Add/Remove Programs
	if v, _, err := k.GetIntegerValue("SystemComponent"); err == nil && v == 1 {
		return true
	}
	// ReleaseType "Update" or "Hotfix" are patches, not installed apps
	if rt, _, err := k.GetStringValue("ReleaseType"); err == nil {
		rt = strings.ToLower(strings.TrimSpace(rt))
		if rt == "update" || rt == "hotfix" || rt == "security update" {
			return true
		}
	}
	return false
}
