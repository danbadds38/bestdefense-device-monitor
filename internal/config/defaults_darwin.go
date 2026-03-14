//go:build darwin

package config

func dataDir() string {
	return "/Library/Application Support/BestDefense"
}
