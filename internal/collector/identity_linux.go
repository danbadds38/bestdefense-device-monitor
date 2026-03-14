//go:build linux

package collector

import (
	"net"
	"os"
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

		// Hardware UUID from DMI (requires root; graceful fallback)
		if data, err := os.ReadFile("/sys/class/dmi/id/product_uuid"); err == nil {
			result.HardwareUUID = strings.TrimSpace(string(data))
		}

		// Serial number from DMI
		if data, err := os.ReadFile("/sys/class/dmi/id/product_serial"); err == nil {
			serial := strings.TrimSpace(string(data))
			// Filter out placeholder values
			if serial != "" && serial != "Not Specified" && serial != "None" {
				result.SerialNumber = serial
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
