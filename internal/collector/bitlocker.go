//go:build windows

package collector

import (
	"fmt"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// BitLocker protection status values from Win32_EncryptableVolume
var bitlockerProtectionStatus = map[int]string{
	0: "unprotected",
	1: "protected",
	2: "unknown",
}

// BitLocker lock status values
var bitlockerLockStatus = map[int]string{
	0: "unlocked",
	1: "locked",
}

// BitLocker conversion status values
var bitlockerConversionStatus = map[int]string{
	0: "fully_decrypted",
	1: "fully_encrypted",
	2: "encryption_in_progress",
	3: "decryption_in_progress",
	4: "encryption_paused",
	5: "decryption_paused",
}

// BitLocker encryption method values
var bitlockerEncryptionMethod = map[int]string{
	0: "none",
	1: "Aes128WithDiffuser",
	2: "Aes256WithDiffuser",
	3: "Aes128",
	4: "Aes256",
	6: "XtsAes128",
	7: "XtsAes256",
}

func collectBitLocker() (result reporter.BitLockerInfo) {
	err := safeCollect("bitlocker", func() error {
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

		// BitLocker lives in a different WMI namespace
		serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil,
			`root\cimv2\security\microsoftvolumeencryption`)
		if err != nil {
			return fmt.Errorf("connecting to BitLocker WMI namespace (requires elevation): %w", err)
		}
		service := serviceRaw.ToIDispatch()
		defer service.Release()

		qResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT DriveLetter, ProtectionStatus, LockStatus, EncryptionMethod, ConversionStatus, EncryptionPercentage FROM Win32_EncryptableVolume")
		if err != nil {
			return fmt.Errorf("querying Win32_EncryptableVolume: %w", err)
		}
		volEnum := qResult.ToIDispatch()
		defer volEnum.Release()

		oleutil.ForEach(volEnum, func(v *ole.VARIANT) error {
			item := v.ToIDispatch()
			defer item.Release()

			var drive reporter.BitLockerDrive

			if dl, err := oleutil.GetProperty(item, "DriveLetter"); err == nil {
				drive.DriveLetter = strings.TrimSpace(dl.ToString())
			}
			if ps, err := oleutil.GetProperty(item, "ProtectionStatus"); err == nil {
				code := int(ps.Val)
				if label, ok := bitlockerProtectionStatus[code]; ok {
					drive.ProtectionStatus = label
				} else {
					drive.ProtectionStatus = fmt.Sprintf("unknown_%d", code)
				}
			}
			if ls, err := oleutil.GetProperty(item, "LockStatus"); err == nil {
				code := int(ls.Val)
				if label, ok := bitlockerLockStatus[code]; ok {
					drive.LockStatus = label
				} else {
					drive.LockStatus = fmt.Sprintf("unknown_%d", code)
				}
			}
			if em, err := oleutil.GetProperty(item, "EncryptionMethod"); err == nil {
				code := int(em.Val)
				if label, ok := bitlockerEncryptionMethod[code]; ok {
					drive.EncryptionMethod = label
				} else {
					drive.EncryptionMethod = fmt.Sprintf("method_%d", code)
				}
			}
			if cs, err := oleutil.GetProperty(item, "ConversionStatus"); err == nil {
				code := int(cs.Val)
				if label, ok := bitlockerConversionStatus[code]; ok {
					drive.ConversionStatus = label
				} else {
					drive.ConversionStatus = fmt.Sprintf("status_%d", code)
				}
			}
			if ep, err := oleutil.GetProperty(item, "EncryptionPercentage"); err == nil {
				drive.PercentageEncrypted = int(ep.Val)
			}

			result.Drives = append(result.Drives, drive)
			return nil
		})

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
