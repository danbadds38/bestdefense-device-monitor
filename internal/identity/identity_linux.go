//go:build linux

package identity

import "os"

func writeKeyFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
