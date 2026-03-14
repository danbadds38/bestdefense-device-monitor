//go:build windows

package collector

import (
	"fmt"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func collectHardware() (result reporter.HardwareInfo) {
	err := safeCollect("hardware", func() error {
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

		// CPU
		cpuResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT Name, NumberOfCores, NumberOfLogicalProcessors FROM Win32_Processor")
		if err == nil {
			cpuEnum := cpuResult.ToIDispatch()
			defer cpuEnum.Release()
			oleutil.ForEach(cpuEnum, func(v *ole.VARIANT) error {
				item := v.ToIDispatch()
				defer item.Release()
				if name, err := oleutil.GetProperty(item, "Name"); err == nil {
					result.CPUName = strings.TrimSpace(name.ToString())
				}
				if cores, err := oleutil.GetProperty(item, "NumberOfCores"); err == nil {
					result.CPUCores = int(cores.Val)
				}
				if lp, err := oleutil.GetProperty(item, "NumberOfLogicalProcessors"); err == nil {
					result.CPULogicalProcs = int(lp.Val)
				}
				return nil
			})
		}

		// RAM from Win32_OperatingSystem (TotalVisibleMemorySize is in KB)
		memResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT TotalVisibleMemorySize FROM Win32_OperatingSystem")
		if err == nil {
			memEnum := memResult.ToIDispatch()
			defer memEnum.Release()
			oleutil.ForEach(memEnum, func(v *ole.VARIANT) error {
				item := v.ToIDispatch()
				defer item.Release()
				if mem, err := oleutil.GetProperty(item, "TotalVisibleMemorySize"); err == nil {
					result.RAMTotalBytes = int64(mem.Val) * 1024
				}
				return nil
			})
		}

		// Disks
		diskResult, err := oleutil.CallMethod(service, "ExecQuery",
			"SELECT DeviceID, Model, Size, MediaType, InterfaceType FROM Win32_DiskDrive")
		if err == nil {
			diskEnum := diskResult.ToIDispatch()
			defer diskEnum.Release()
			oleutil.ForEach(diskEnum, func(v *ole.VARIANT) error {
				item := v.ToIDispatch()
				defer item.Release()

				var disk reporter.DiskInfo
				if id, err := oleutil.GetProperty(item, "DeviceID"); err == nil {
					disk.DeviceID = strings.TrimSpace(id.ToString())
				}
				if model, err := oleutil.GetProperty(item, "Model"); err == nil {
					disk.Model = strings.TrimSpace(model.ToString())
				}
				if size, err := oleutil.GetProperty(item, "Size"); err == nil {
					fmt.Sscanf(size.ToString(), "%d", &disk.SizeBytes)
				}
				if mt, err := oleutil.GetProperty(item, "MediaType"); err == nil {
					mt := strings.TrimSpace(mt.ToString())
					switch {
					case strings.Contains(strings.ToUpper(mt), "SSD") ||
						strings.Contains(strings.ToUpper(mt), "SOLID"):
						disk.MediaType = "SSD"
					case strings.Contains(strings.ToUpper(mt), "FIXED"):
						disk.MediaType = "HDD"
					default:
						disk.MediaType = mt
					}
				}
				if it, err := oleutil.GetProperty(item, "InterfaceType"); err == nil {
					disk.InterfaceType = strings.TrimSpace(it.ToString())
				}
				result.Disks = append(result.Disks, disk)
				return nil
			})
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
