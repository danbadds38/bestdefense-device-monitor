//go:build linux

package collector

import (
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectFirewall() (result reporter.FirewallInfo) {
	err := safeCollect("firewall", func() error {
		enabled := false

		// Try ufw first (Debian/Ubuntu)
		if out, err := exec.Command("ufw", "status").Output(); err == nil {
			status := strings.ToLower(string(out))
			enabled = strings.Contains(status, "status: active")
		} else if out, err := exec.Command("firewall-cmd", "--state").Output(); err == nil {
			// Try firewalld (RHEL/Fedora/CentOS)
			enabled = strings.TrimSpace(strings.ToLower(string(out))) == "running"
		} else {
			// Fall back to iptables — check if any non-default rules exist
			if out, err := exec.Command("iptables", "-L", "-n", "--line-numbers").Output(); err == nil {
				lines := strings.Split(string(out), "\n")
				ruleCount := 0
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "Chain") || strings.HasPrefix(line, "target") || strings.HasPrefix(line, "num") {
						continue
					}
					ruleCount++
				}
				enabled = ruleCount > 0
			}
		}

		// Linux has a single "public" profile concept
		result.Profiles.Public = reporter.FirewallProfile{
			Enabled:               enabled,
			DefaultInboundAction:  "block",
			DefaultOutboundAction: "allow",
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
