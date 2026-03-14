//go:build windows

package service

import (
	"fmt"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const firewallRuleName = "BestDefense Device Monitor"

// addFirewallRule adds a Windows Firewall outbound allow rule for the agent binary.
// This prevents corporate firewalls with outbound filtering from silently blocking check-ins.
func addFirewallRule(exePath string) {
	if err := addFWRule(exePath); err != nil {
		// Non-fatal — log but don't fail installation
		fmt.Printf("Warning: could not add firewall rule: %v\n", err)
	}
}

func addFWRule(exePath string) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		// may already be init'd
	}
	defer ole.CoUninitialize()

	policyUnk, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
	if err != nil {
		return fmt.Errorf("creating FwPolicy2: %w", err)
	}
	defer policyUnk.Release()

	policy, err := policyUnk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer policy.Release()

	rulesRaw, err := oleutil.GetProperty(policy, "Rules")
	if err != nil {
		return fmt.Errorf("getting rules: %w", err)
	}
	rules := rulesRaw.ToIDispatch()
	defer rules.Release()

	// Create the rule object
	ruleUnk, err := oleutil.CreateObject("HNetCfg.FWRule")
	if err != nil {
		return fmt.Errorf("creating FWRule: %w", err)
	}
	defer ruleUnk.Release()

	rule, err := ruleUnk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer rule.Release()

	props := map[string]interface{}{
		"Name":        firewallRuleName,
		"Description": "Allow BestDefense Device Monitor to report to app.bestdefense.io",
		"ApplicationName": exePath,
		"Action":      1,   // NET_FW_ACTION_ALLOW
		"Direction":   2,   // NET_FW_RULE_DIR_OUT
		"Enabled":     true,
		"Protocol":    6,   // TCP
	}

	for k, v := range props {
		if _, err := oleutil.PutProperty(rule, k, v); err != nil {
			return fmt.Errorf("setting rule.%s: %w", k, err)
		}
	}

	if _, err := oleutil.CallMethod(rules, "Add", rule); err != nil {
		return fmt.Errorf("adding rule: %w", err)
	}

	return nil
}

// removeFirewallRule removes the agent's firewall rule.
func removeFirewallRule() {
	if err := removeFWRule(); err != nil {
		fmt.Printf("Warning: could not remove firewall rule: %v\n", err)
	}
}

func removeFWRule() error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		// may already be init'd
	}
	defer ole.CoUninitialize()

	policyUnk, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
	if err != nil {
		return err
	}
	defer policyUnk.Release()

	policy, err := policyUnk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer policy.Release()

	rulesRaw, err := oleutil.GetProperty(policy, "Rules")
	if err != nil {
		return err
	}
	rules := rulesRaw.ToIDispatch()
	defer rules.Release()

	_, err = oleutil.CallMethod(rules, "Remove", firewallRuleName)
	return err
}
