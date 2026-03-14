//go:build darwin

package collector

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectHardware() (result reporter.HardwareInfo) {
	err := safeCollect("hardware", func() error {
		collectCPUAndRAM(&result)
		collectDisks(&result)
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

func collectCPUAndRAM(result *reporter.HardwareInfo) {
	out, err := exec.Command("system_profiler", "SPHardwareDataType").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Chip:"):
			// Apple Silicon: "Chip: Apple M3 Pro"
			result.CPUName = strings.TrimSpace(strings.TrimPrefix(line, "Chip:"))
		case strings.HasPrefix(line, "Processor Name:"):
			// Intel: "Processor Name: Intel Core i9"
			if result.CPUName == "" {
				result.CPUName = strings.TrimSpace(strings.TrimPrefix(line, "Processor Name:"))
			}
		case strings.HasPrefix(line, "Total Number of Cores:"):
			// "Total Number of Cores: 12 (8 performance and 4 efficiency)"
			val := strings.TrimSpace(strings.TrimPrefix(line, "Total Number of Cores:"))
			// Take only the number before the first space
			if parts := strings.Fields(val); len(parts) > 0 {
				if n, err := strconv.Atoi(parts[0]); err == nil {
					result.CPUCores = n
					result.CPULogicalProcs = n
				}
			}
		case strings.HasPrefix(line, "Memory:"):
			// "Memory: 16 GB"
			val := strings.TrimSpace(strings.TrimPrefix(line, "Memory:"))
			result.RAMTotalBytes = parseMemoryString(val)
		}
	}
}

// parseMemoryString converts "16 GB" or "8 GB" to bytes.
func parseMemoryString(s string) int64 {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(fields[1]) {
	case "GB":
		return int64(val * 1024 * 1024 * 1024)
	case "MB":
		return int64(val * 1024 * 1024)
	case "TB":
		return int64(val * 1024 * 1024 * 1024 * 1024)
	}
	return 0
}

// spStorageDataType is the JSON shape of system_profiler SPStorageDataType -json
type spStorageOutput struct {
	SPStorageDataType []spStorageItem `json:"SPStorageDataType"`
}

type spStorageItem struct {
	Name       string `json:"_name"`
	SizeBytes  int64  `json:"size_in_bytes"`
	MediaType  string `json:"spdisk_type"`
	DeviceID   string `json:"bsd_name"`
}

func collectDisks(result *reporter.HardwareInfo) {
	out, err := exec.Command("system_profiler", "SPStorageDataType", "-json").Output()
	if err != nil {
		return
	}
	var sp spStorageOutput
	if err := json.Unmarshal(out, &sp); err != nil {
		return
	}
	for _, item := range sp.SPStorageDataType {
		disk := reporter.DiskInfo{
			DeviceID:  item.DeviceID,
			Model:     item.Name,
			SizeBytes: item.SizeBytes,
			MediaType: item.MediaType,
		}
		result.Disks = append(result.Disks, disk)
	}
}
