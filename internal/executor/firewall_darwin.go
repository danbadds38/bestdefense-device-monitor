//go:build darwin

package executor

import (
	"os/exec"
	"strings"
)

func init() { enableFirewall = darwinEnableFirewall }

func darwinEnableFirewall() (string, error) {
	out, err := exec.Command(
		"/usr/libexec/ApplicationFirewall/socketfilterfw",
		"--setglobalstate", "on",
	).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
