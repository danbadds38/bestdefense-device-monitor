//go:build linux

package collector

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectHardware() (result reporter.HardwareInfo) {
	err := safeCollect("hardware", func() error {
		// CPU info from /proc/cpuinfo
		if f, err := os.Open("/proc/cpuinfo"); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			logicalProcs := 0
			for scanner.Scan() {
				line := scanner.Text()
				key, val, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				key = strings.TrimSpace(key)
				val = strings.TrimSpace(val)
				switch key {
				case "model name":
					if result.CPUName == "" {
						result.CPUName = val
					}
				case "processor":
					logicalProcs++
				}
			}
			result.CPULogicalProcs = logicalProcs
		}

		// Physical cores via lscpu
		if out, err := exec.Command("lscpu").Output(); err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				line := scanner.Text()
				key, val, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				if strings.TrimSpace(key) == "Core(s) per socket" {
					if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
						result.CPUCores = n
					}
				}
			}
		}
		if result.CPUCores == 0 {
			result.CPUCores = result.CPULogicalProcs
		}

		// RAM from /proc/meminfo
		if f, err := os.Open("/proc/meminfo"); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
							result.RAMTotalBytes = kb * 1024
						}
					}
					break
				}
			}
		}

		// Disks via lsblk JSON output
		result.Disks = collectLinuxDisks()

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name      string `json:"name"`
	Size      string `json:"size"`
	Type      string `json:"type"`
	Model     string `json:"model"`
	Rota      bool   `json:"rota"` // true = HDD, false = SSD
	Tran      string `json:"tran"` // "sata", "nvme", "usb", etc.
}

func collectLinuxDisks() []reporter.DiskInfo {
	var disks []reporter.DiskInfo

	out, err := exec.Command("lsblk", "-J", "-b", "-d", "-o", "NAME,SIZE,TYPE,MODEL,ROTA,TRAN").Output()
	if err != nil {
		return disks
	}

	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return disks
	}

	for _, dev := range parsed.Blockdevices {
		if dev.Type != "disk" {
			continue
		}

		mediaType := "HDD"
		if !dev.Rota {
			mediaType = "SSD"
		}
		interfaceType := strings.ToUpper(dev.Tran)
		if interfaceType == "" {
			interfaceType = "Unknown"
		}

		// lsblk -b returns size as string of bytes
		var sizeBytes int64
		if n, err := strconv.ParseInt(strings.TrimSpace(dev.Size), 10, 64); err == nil {
			sizeBytes = n
		}

		disks = append(disks, reporter.DiskInfo{
			DeviceID:      "/dev/" + dev.Name,
			Model:         strings.TrimSpace(dev.Model),
			SizeBytes:     sizeBytes,
			MediaType:     mediaType,
			InterfaceType: interfaceType,
		})
	}

	return disks
}
