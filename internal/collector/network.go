package collector

import (
	"net"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectNetwork() (result reporter.NetworkInfo) {
	err := safeCollect("network", func() error {
		ifaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range ifaces {
			ni := reporter.NetworkInterface{
				Name:       iface.Name,
				MACAddress: iface.HardwareAddr.String(),
				IsUp:       iface.Flags&net.FlagUp != 0,
			}

			// Determine type from flags / name
			if iface.Flags&net.FlagLoopback != 0 {
				ni.Type = "loopback"
			} else if strings.Contains(strings.ToLower(iface.Name), "wi-fi") ||
				strings.Contains(strings.ToLower(iface.Name), "wireless") ||
				strings.Contains(strings.ToLower(iface.Name), "wlan") {
				ni.Type = "wifi"
			} else {
				ni.Type = "ethernet"
			}

			addrs, err := iface.Addrs()
			if err == nil {
				for _, addr := range addrs {
					// Strip CIDR suffix
					ip := addr.String()
					if idx := strings.Index(ip, "/"); idx >= 0 {
						ip = ip[:idx]
					}
					ni.IPAddresses = append(ni.IPAddresses, ip)
				}
			}

			result.Interfaces = append(result.Interfaces, ni)
		}
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
