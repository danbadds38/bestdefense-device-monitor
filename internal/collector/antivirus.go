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

func collectAntivirus() (result reporter.AntivirusInfo) {
	err := safeCollect("antivirus", func() error {
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

		// Windows Defender status is in root\Microsoft\Windows\Defender namespace
		serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil,
			`root\Microsoft\Windows\Defender`)
		if err != nil {
			return fmt.Errorf("connecting to Defender WMI namespace (requires elevation): %w", err)
		}
		service := serviceRaw.ToIDispatch()
		defer service.Release()

		qResult, err := oleutil.CallMethod(service, "ExecQuery",
			`SELECT AMServiceEnabled, AntispywareEnabled, AntivirusEnabled,
			        BehaviorMonitorEnabled, OnAccessProtectionEnabled,
			        RealTimeProtectionEnabled, AntispywareSignatureVersion,
			        AntispywareSignatureLastUpdated, AntivirusSignatureVersion,
			        AntivirusSignatureLastUpdated
			 FROM MSFT_MpComputerStatus`)
		if err != nil {
			return fmt.Errorf("querying MSFT_MpComputerStatus: %w", err)
		}
		mpEnum := qResult.ToIDispatch()
		defer mpEnum.Release()

		oleutil.ForEach(mpEnum, func(v *ole.VARIANT) error {
			item := v.ToIDispatch()
			defer item.Release()

			if p, err := oleutil.GetProperty(item, "AMServiceEnabled"); err == nil {
				result.AMServiceEnabled = p.Val != 0
				result.WindowsDefenderEnabled = p.Val != 0
			}
			if p, err := oleutil.GetProperty(item, "RealTimeProtectionEnabled"); err == nil {
				result.RealtimeProtectionEnabled = p.Val != 0
			}
			if p, err := oleutil.GetProperty(item, "AntispywareEnabled"); err == nil {
				result.AntispywareEnabled = p.Val != 0
			}
			if p, err := oleutil.GetProperty(item, "BehaviorMonitorEnabled"); err == nil {
				result.BehaviorMonitorEnabled = p.Val != 0
			}
			if p, err := oleutil.GetProperty(item, "OnAccessProtectionEnabled"); err == nil {
				result.OnAccessProtectionEnabled = p.Val != 0
			}

			// Prefer antivirus signature version; fall back to antispyware
			if p, err := oleutil.GetProperty(item, "AntivirusSignatureVersion"); err == nil {
				result.DefinitionVersion = strings.TrimSpace(p.ToString())
			}
			if result.DefinitionVersion == "" {
				if p, err := oleutil.GetProperty(item, "AntispywareSignatureVersion"); err == nil {
					result.DefinitionVersion = strings.TrimSpace(p.ToString())
				}
			}

			// Definition last updated date
			if p, err := oleutil.GetProperty(item, "AntivirusSignatureLastUpdated"); err == nil {
				raw := strings.TrimSpace(p.ToString())
				if t, err := parseWMIDate(raw); err == nil {
					result.DefinitionDate = &t
				}
			}

			// Determine overall product status
			if result.WindowsDefenderEnabled && result.RealtimeProtectionEnabled {
				if result.DefinitionDate != nil && time.Since(*result.DefinitionDate) < 7*24*time.Hour {
					result.ProductStatus = "up_to_date"
				} else {
					result.ProductStatus = "definitions_outdated"
				}
			} else {
				result.ProductStatus = "disabled"
			}

			return nil
		})

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
