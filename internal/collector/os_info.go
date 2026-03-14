//go:build windows

package collector

import (
	"fmt"
	"strings"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"
)

func collectOSInfo() (result reporter.OSInfo) {
	err := safeCollect("os_info", func() error {
		// Registry for display info (more reliable than WMI for display version)
		k, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
		if err == nil {
			defer k.Close()
			if v, _, err := k.GetStringValue("ProductName"); err == nil {
				result.Name = v
			}
			if v, _, err := k.GetStringValue("CurrentBuildNumber"); err == nil {
				result.BuildNumber = v
			}
			if v, _, err := k.GetStringValue("DisplayVersion"); err == nil {
				result.DisplayVersion = v
			} else if v, _, err := k.GetStringValue("ReleaseId"); err == nil {
				result.DisplayVersion = v
			}
			if v, _, err := k.GetStringValue("CurrentVersion"); err == nil {
				result.Version = v + "." + result.BuildNumber
			}
			if v, _, err := k.GetStringValue("RegisteredOwner"); err == nil {
				result.RegisteredOwner = v
			}
			if v, _, err := k.GetStringValue("RegisteredOrganization"); err == nil {
				result.RegisteredOrganization = v
			}
			// Install date is a DWORD (Unix timestamp)
			if v, _, err := k.GetIntegerValue("InstallDate"); err == nil {
				t := time.Unix(int64(v), 0).UTC()
				result.InstallDate = &t
			}
		}

		// Architecture
		k2, err := registry.OpenKey(registry.LOCAL_MACHINE,
			`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
			registry.QUERY_VALUE)
		if err == nil {
			defer k2.Close()
			if v, _, err := k2.GetStringValue("PROCESSOR_ARCHITECTURE"); err == nil {
				switch strings.ToUpper(v) {
				case "AMD64":
					result.Architecture = "x86_64"
				case "X86":
					result.Architecture = "x86"
				case "ARM64":
					result.Architecture = "arm64"
				default:
					result.Architecture = v
				}
			}
		}

		// Full version from WMI Win32_OperatingSystem
		if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
			// may already be init'd
		}
		defer ole.CoUninitialize()

		unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
		if err != nil {
			return fmt.Errorf("WMI locator: %w", err)
		}
		defer unknown.Release()

		wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return fmt.Errorf("WMI interface: %w", err)
		}
		defer wmi.Release()

		serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, "root\\cimv2")
		if err != nil {
			return fmt.Errorf("WMI connect: %w", err)
		}
		service := serviceRaw.ToIDispatch()
		defer service.Release()

		qResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT Version FROM Win32_OperatingSystem")
		if err != nil {
			return fmt.Errorf("WMI OS query: %w", err)
		}
		osEnum := qResult.ToIDispatch()
		defer osEnum.Release()

		oleutil.ForEach(osEnum, func(v *ole.VARIANT) error {
			item := v.ToIDispatch()
			defer item.Release()
			if ver, err := oleutil.GetProperty(item, "Version"); err == nil {
				result.Version = strings.TrimSpace(ver.ToString())
			}
			return nil
		})

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
