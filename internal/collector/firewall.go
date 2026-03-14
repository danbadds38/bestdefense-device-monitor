//go:build windows

package collector

import (
	"fmt"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// NET_FW_ACTION values
const (
	netFwActionBlock = 0
	netFwActionAllow = 1
)

func actionStr(v int) string {
	if v == netFwActionAllow {
		return "allow"
	}
	return "block"
}

func collectFirewall() (result reporter.FirewallInfo) {
	err := safeCollect("firewall", func() error {
		if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
			// may already be init'd
		}
		defer ole.CoUninitialize()

		// Use HNetCfg.FwPolicy2 COM object — more reliable than WMI or netsh parsing
		unknown, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
		if err != nil {
			return fmt.Errorf("creating HNetCfg.FwPolicy2: %w", err)
		}
		defer unknown.Release()

		fw, err := unknown.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return fmt.Errorf("querying firewall interface: %w", err)
		}
		defer fw.Release()

		// Profile types: 1=domain, 2=private, 4=public
		profileTypes := []struct {
			code   int
			target *reporter.FirewallProfile
		}{
			{1, &result.Profiles.Domain},
			{2, &result.Profiles.Private},
			{4, &result.Profiles.Public},
		}

		for _, pt := range profileTypes {
			enabled, err := oleutil.CallMethod(fw, "FirewallEnabled", int32(pt.code))
			if err == nil {
				pt.target.Enabled = enabled.Val != 0
			}

			inbound, err := oleutil.CallMethod(fw, "DefaultInboundAction", int32(pt.code))
			if err == nil {
				pt.target.DefaultInboundAction = actionStr(int(inbound.Val))
			} else {
				pt.target.DefaultInboundAction = "block" // safe default
			}

			outbound, err := oleutil.CallMethod(fw, "DefaultOutboundAction", int32(pt.code))
			if err == nil {
				pt.target.DefaultOutboundAction = actionStr(int(outbound.Val))
			} else {
				pt.target.DefaultOutboundAction = "allow" // safe default
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
