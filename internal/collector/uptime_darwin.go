//go:build darwin

package collector

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectSystemHealth() (result reporter.SystemHealthInfo) {
	err := safeCollect("system_health", func() error {
		// sysctl kern.boottime returns something like:
		// { sec = 1710410000, usec = 0 } Thu Mar 14 06:13:20 2026
		out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
		if err != nil {
			return fmt.Errorf("sysctl kern.boottime: %w", err)
		}

		bootTime, err := parseKernBoottime(strings.TrimSpace(string(out)))
		if err != nil {
			return err
		}

		result.LastRebootTime = &bootTime
		result.UptimeHours = time.Since(bootTime).Hours()
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// parseKernBoottime parses "{ sec = 1710410000, usec = 0 } ..." into a time.Time.
func parseKernBoottime(s string) (time.Time, error) {
	// Find "sec = <number>"
	idx := strings.Index(s, "sec = ")
	if idx < 0 {
		return time.Time{}, fmt.Errorf("unexpected kern.boottime format: %q", s)
	}
	rest := s[idx+len("sec = "):]
	end := strings.IndexAny(rest, ", }")
	if end < 0 {
		end = len(rest)
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(rest[:end]), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing kern.boottime sec: %w", err)
	}
	return time.Unix(sec, 0).UTC(), nil
}
