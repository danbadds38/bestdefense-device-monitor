//go:build linux

package collector

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectOSInfo() (result reporter.OSInfo) {
	err := safeCollect("os_info", func() error {
		result.Name = "Linux"

		// Parse /etc/os-release for distro name and version
		if f, err := os.Open("/etc/os-release"); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				key, val, ok := strings.Cut(line, "=")
				if !ok {
					continue
				}
				val = strings.Trim(val, `"`)
				switch key {
				case "PRETTY_NAME":
					result.Name = val
				case "VERSION_ID":
					result.Version = val
				case "BUILD_ID":
					result.BuildNumber = val
				}
			}
		}

		// Kernel version via uname -r
		if out, err := exec.Command("uname", "-r").Output(); err == nil {
			result.BuildNumber = strings.TrimSpace(string(out))
		}

		// Architecture via uname -m
		if out, err := exec.Command("uname", "-m").Output(); err == nil {
			result.Architecture = strings.TrimSpace(string(out))
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
