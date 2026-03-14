//go:build linux

package collector

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectSystemHealth() (result reporter.SystemHealthInfo) {
	err := safeCollect("system_health", func() error {
		// /proc/uptime contains: <uptime_seconds> <idle_seconds>
		data, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return fmt.Errorf("reading /proc/uptime: %w", err)
		}

		fields := strings.Fields(string(data))
		if len(fields) < 1 {
			return fmt.Errorf("unexpected /proc/uptime format: %q", string(data))
		}

		uptimeSecs, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return fmt.Errorf("parsing uptime seconds: %w", err)
		}

		result.UptimeHours = uptimeSecs / 3600
		bootTime := time.Now().Add(-time.Duration(uptimeSecs * float64(time.Second))).UTC()
		result.LastRebootTime = &bootTime

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
