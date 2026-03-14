//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

const socketfilterfw = "/usr/libexec/ApplicationFirewall/socketfilterfw"

func collectFirewall() (result reporter.FirewallInfo) {
	err := safeCollect("firewall", func() error {
		out, err := exec.Command(socketfilterfw, "--getglobalstate").Output()
		if err != nil {
			return err
		}

		status := strings.ToLower(strings.TrimSpace(string(out)))
		enabled := strings.Contains(status, "enabled") || strings.Contains(status, "state = 1")

		// macOS has a single application firewall (no domain/private/public split).
		// Populate "public" profile; leave domain and private as zero-value (disabled).
		profile := reporter.FirewallProfile{
			Enabled:               enabled,
			DefaultInboundAction:  "block", // macOS ALF blocks inbound by default when enabled
			DefaultOutboundAction: "allow",
		}

		result.Profiles = reporter.FirewallProfiles{
			Public: profile,
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
