//go:build darwin

package collector

import (
	"net"
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectIdentity() (result reporter.DeviceIdentity) {
	err := safeCollect("identity", func() error {
		// Hostname
		if out, err := exec.Command("hostname").Output(); err == nil {
			result.Hostname = strings.TrimSpace(string(out))
			result.ComputerName = result.Hostname
		}

		// Hardware UUID and Serial Number from system_profiler
		if out, err := exec.Command("system_profiler", "SPHardwareDataType").Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Serial Number") {
					if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
						result.SerialNumber = strings.TrimSpace(parts[1])
					}
				} else if strings.HasPrefix(line, "Hardware UUID") {
					if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
						result.HardwareUUID = strings.TrimSpace(parts[1])
					}
				} else if strings.HasPrefix(line, "Provisioning UDID") {
					if result.HardwareUUID == "" {
						if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
							result.HardwareUUID = strings.TrimSpace(parts[1])
						}
					}
				}
			}
		}

		// MAC addresses from network interfaces
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range ifaces {
				if mac := iface.HardwareAddr.String(); mac != "" {
					result.MACAddresses = append(result.MACAddresses, mac)
				}
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
