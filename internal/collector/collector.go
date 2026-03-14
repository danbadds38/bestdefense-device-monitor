//go:build windows

package collector

import (
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

// Collect runs all checks and assembles a DeviceReport.
// Individual check failures are captured in each section's CollectionError field
// and in the top-level CheckErrors slice — they do not abort the collection.
func Collect(cfg *config.Config) *reporter.DeviceReport {
	report := &reporter.DeviceReport{
		SchemaVersion:   "1",
		RegistrationKey: cfg.RegistrationKey,
		AgentVersion:    cfg.AgentVersion,
		CollectedAt:     time.Now().UTC(),
	}

	type check struct {
		name string
		run  func()
	}

	checks := []check{
		{"identity", func() {
			report.DeviceIdentity = collectIdentity()
			if report.DeviceIdentity.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "identity", Error: *report.DeviceIdentity.CollectionError,
				})
			}
		}},
		{"os_info", func() {
			report.OS = collectOSInfo()
			if report.OS.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "os_info", Error: *report.OS.CollectionError,
				})
			}
		}},
		{"hardware", func() {
			report.Hardware = collectHardware()
			if report.Hardware.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "hardware", Error: *report.Hardware.CollectionError,
				})
			}
		}},
		{"bitlocker", func() {
			report.BitLocker = collectBitLocker()
			if report.BitLocker.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "bitlocker", Error: *report.BitLocker.CollectionError,
				})
			}
		}},
		{"antivirus", func() {
			report.Antivirus = collectAntivirus()
			if report.Antivirus.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "antivirus", Error: *report.Antivirus.CollectionError,
				})
			}
		}},
		{"firewall", func() {
			report.Firewall = collectFirewall()
			if report.Firewall.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "firewall", Error: *report.Firewall.CollectionError,
				})
			}
		}},
		{"screen_lock", func() {
			report.ScreenLock = collectScreenLock()
			if report.ScreenLock.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "screen_lock", Error: *report.ScreenLock.CollectionError,
				})
			}
		}},
		{"windows_update", func() {
			report.WindowsUpdate = collectWindowsUpdate()
			if report.WindowsUpdate.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "windows_update", Error: *report.WindowsUpdate.CollectionError,
				})
			}
		}},
		{"applications", func() {
			report.InstalledApps = collectApplications()
			if report.InstalledApps.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "applications", Error: *report.InstalledApps.CollectionError,
				})
			}
		}},
		{"local_users", func() {
			report.LocalUsers = collectLocalUsers()
			if report.LocalUsers.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "local_users", Error: *report.LocalUsers.CollectionError,
				})
			}
		}},
		{"password_policy", func() {
			report.PasswordPolicy = collectPasswordPolicy()
			if report.PasswordPolicy.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "password_policy", Error: *report.PasswordPolicy.CollectionError,
				})
			}
		}},
		{"network", func() {
			report.NetworkInterfaces = collectNetwork()
			if report.NetworkInterfaces.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "network", Error: *report.NetworkInterfaces.CollectionError,
				})
			}
		}},
		{"system_health", func() {
			report.SystemHealth = collectSystemHealth()
			if report.SystemHealth.CollectionError != nil {
				report.CheckErrors = append(report.CheckErrors, reporter.CheckError{
					Check: "system_health", Error: *report.SystemHealth.CollectionError,
				})
			}
		}},
	}

	for _, c := range checks {
		c.run()
	}

	return report
}
