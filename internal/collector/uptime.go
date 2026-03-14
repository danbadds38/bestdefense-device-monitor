//go:build windows

package collector

import (
	"fmt"
	"strings"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func collectSystemHealth() (result reporter.SystemHealthInfo) {
	err := safeCollect("system_health", func() error {
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
			"SELECT LastBootUpTime FROM Win32_OperatingSystem")
		if err != nil {
			return fmt.Errorf("WMI OS query: %w", err)
		}
		osEnum := qResult.ToIDispatch()
		defer osEnum.Release()

		oleutil.ForEach(osEnum, func(v *ole.VARIANT) error {
			item := v.ToIDispatch()
			defer item.Release()
			if lbt, err := oleutil.GetProperty(item, "LastBootUpTime"); err == nil {
				raw := strings.TrimSpace(lbt.ToString())
				// WMI datetime format: "20260314084500.000000+000"
				t, err := parseWMIDate(raw)
				if err == nil {
					result.LastRebootTime = &t
					result.UptimeHours = time.Since(t).Hours()
				}
			}
			return nil
		})

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// parseWMIDate parses a WMI datetime string (yyyymmddHHMMSS.uuuuuu±UUU)
func parseWMIDate(s string) (time.Time, error) {
	if len(s) < 14 {
		return time.Time{}, fmt.Errorf("datetime too short: %q", s)
	}
	// Parse as UTC ignoring offset for simplicity
	t, err := time.Parse("20060102150405", s[:14])
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing WMI date %q: %w", s, err)
	}
	return t.UTC(), nil
}
