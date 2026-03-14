//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectDiskEncryption() (result reporter.DiskEncryptionInfo) {
	err := safeCollect("disk_encryption", func() error {
		out, err := exec.Command("fdesetup", "status").Output()
		if err != nil {
			// fdesetup requires root to give full info; degrade gracefully
			result.CollectionError = strPtr("fdesetup requires root to report FileVault status")
			return nil
		}

		status := strings.TrimSpace(string(out))
		drive := reporter.EncryptedDriveInfo{
			DriveLetter:  "/",
			LockStatus:   "unlocked", // system is running, so the volume is unlocked
			ConversionStatus: "fully_encrypted",
			PercentageEncrypted: 100,
		}

		if strings.HasPrefix(status, "FileVault is On") {
			drive.ProtectionStatus = "protected"
			drive.EncryptionMethod = "FileVault"
		} else if strings.HasPrefix(status, "FileVault is Off") {
			drive.ProtectionStatus = "unprotected"
			drive.EncryptionMethod = "none"
			drive.ConversionStatus = "fully_decrypted"
			drive.PercentageEncrypted = 0
		} else {
			drive.ProtectionStatus = "unknown"
			drive.EncryptionMethod = "unknown"
		}

		result.Drives = append(result.Drives, drive)
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
