//go:build windows

package collector

import (
	"golang.org/x/sys/windows/registry"
)

// registryCheckDWORD opens a registry key under HKLM and checks if the named
// DWORD value equals expectedValue. Returns false if the key or value is missing.
func registryCheckDWORD(path, name string, expectedValue uint64) bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(name)
	if err != nil {
		return false
	}
	return v == expectedValue
}
