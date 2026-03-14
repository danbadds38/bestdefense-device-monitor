//go:build linux

package collector

import (
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectDiskEncryption() (result reporter.DiskEncryptionInfo) {
	err := safeCollect("disk_encryption", func() error {
		// Use lsblk to find dm-crypt (LUKS) volumes
		type lsblkDev struct {
			Name     string     `json:"name"`
			Type     string     `json:"type"` // "part", "crypt", "disk", "lvm", etc.
			Fstype   string     `json:"fstype"`
			Children []lsblkDev `json:"children"`
		}
		type lsblkOut struct {
			Blockdevices []lsblkDev `json:"blockdevices"`
		}

		out, err := exec.Command("lsblk", "-J", "-o", "NAME,TYPE,FSTYPE").Output()
		if err != nil {
			return err
		}

		var parsed lsblkOut
		if err := json.Unmarshal(out, &parsed); err != nil {
			return err
		}

		// Walk the device tree looking for encrypted volumes
		var walk func(devs []lsblkDev, parentName string)
		walk = func(devs []lsblkDev, parentName string) {
			for _, dev := range devs {
				if dev.Type == "crypt" || dev.Fstype == "crypto_LUKS" {
					name := dev.Name
					if parentName != "" {
						name = parentName
					}
					result.Drives = append(result.Drives, reporter.EncryptedDriveInfo{
						DriveLetter:         "/dev/" + name,
						ProtectionStatus:    "protected",
						EncryptionMethod:    "LUKS",
						LockStatus:          "unlocked", // it's mounted, so unlocked
						ConversionStatus:    "fully_encrypted",
						PercentageEncrypted: 100,
					})
				} else if dev.Fstype == "crypto_LUKS" {
					// Locked LUKS partition (no dm-crypt child)
					result.Drives = append(result.Drives, reporter.EncryptedDriveInfo{
						DriveLetter:         "/dev/" + dev.Name,
						ProtectionStatus:    "protected",
						EncryptionMethod:    "LUKS",
						LockStatus:          "locked",
						ConversionStatus:    "fully_encrypted",
						PercentageEncrypted: 100,
					})
				}
				if len(dev.Children) > 0 {
					walk(dev.Children, dev.Name)
				}
			}
		}
		walk(parsed.Blockdevices, "")

		// If no encrypted volumes found, report root as unprotected
		if len(result.Drives) == 0 {
			result.Drives = append(result.Drives, reporter.EncryptedDriveInfo{
				DriveLetter:      "/",
				ProtectionStatus: "unprotected",
				EncryptionMethod: "",
				LockStatus:       "unlocked",
			})
		}

		// Refine LUKS version if dmsetup is available
		if out, err := exec.Command("dmsetup", "status").Output(); err == nil {
			for i, drive := range result.Drives {
				if drive.EncryptionMethod == "LUKS" {
					name := strings.TrimPrefix(drive.DriveLetter, "/dev/")
					if strings.Contains(string(out), name) {
						result.Drives[i].EncryptionMethod = "LUKS2"
					}
				}
			}
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
