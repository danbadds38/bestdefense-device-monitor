//go:build linux

package executor

import (
	"os/exec"
	"strings"
)

func init() { enableFirewall = linuxEnableFirewall }

func linuxEnableFirewall() (string, error) {
	out, err := exec.Command("ufw", "--force", "enable").CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
