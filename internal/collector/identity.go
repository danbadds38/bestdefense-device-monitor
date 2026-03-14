//go:build windows

package collector

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func collectIdentity() (result reporter.DeviceIdentity) {
	err := safeCollect("identity", func() error {
		hostname, _ := os.Hostname()
		result.Hostname = hostname
		result.ComputerName = hostname

		// MAC addresses from active non-loopback interfaces
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range ifaces {
				if iface.Flags&net.FlagLoopback != 0 {
					continue
				}
				if len(iface.HardwareAddr) == 0 {
					continue
				}
				result.MACAddresses = append(result.MACAddresses, iface.HardwareAddr.String())
			}
		}

		// Serial number and hardware UUID from WMI
		if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
			// May already be initialized; ignore
		}
		defer ole.CoUninitialize()

		unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
		if err != nil {
			return fmt.Errorf("creating WMI locator: %w", err)
		}
		defer unknown.Release()

		wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return fmt.Errorf("querying WMI interface: %w", err)
		}
		defer wmi.Release()

		serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, "root\\cimv2")
		if err != nil {
			return fmt.Errorf("connecting to WMI: %w", err)
		}
		service := serviceRaw.ToIDispatch()
		defer service.Release()

		// Serial number from Win32_BIOS
		biosResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT SerialNumber FROM Win32_BIOS")
		if err == nil {
			biosEnum := biosResult.ToIDispatch()
			defer biosEnum.Release()
			oleutil.ForEach(biosEnum, func(v *ole.VARIANT) error {
				item := v.ToIDispatch()
				defer item.Release()
				if sn, err := oleutil.GetProperty(item, "SerialNumber"); err == nil {
					result.SerialNumber = strings.TrimSpace(sn.ToString())
				}
				return nil
			})
		}

		// Hardware UUID from Win32_ComputerSystemProduct
		cspResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT UUID FROM Win32_ComputerSystemProduct")
		if err == nil {
			cspEnum := cspResult.ToIDispatch()
			defer cspEnum.Release()
			oleutil.ForEach(cspEnum, func(v *ole.VARIANT) error {
				item := v.ToIDispatch()
				defer item.Release()
				if uuid, err := oleutil.GetProperty(item, "UUID"); err == nil {
					result.HardwareUUID = strings.TrimSpace(uuid.ToString())
				}
				return nil
			})
		}

		// Domain from Win32_ComputerSystem
		csResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT Domain FROM Win32_ComputerSystem")
		if err == nil {
			csEnum := csResult.ToIDispatch()
			defer csEnum.Release()
			oleutil.ForEach(csEnum, func(v *ole.VARIANT) error {
				item := v.ToIDispatch()
				defer item.Release()
				if domain, err := oleutil.GetProperty(item, "Domain"); err == nil {
					result.Domain = strings.TrimSpace(domain.ToString())
				}
				return nil
			})
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
