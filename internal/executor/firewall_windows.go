//go:build windows

package executor

import (
	"os/exec"
	"strings"
)

func init() { enableFirewall = windowsEnableFirewall }

func windowsEnableFirewall() (string, error) {
	out, err := exec.Command(
		"netsh", "advfirewall", "set", "allprofiles", "state", "on",
	).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
