//go:build windows

package collector

import (
	"fmt"
	"unsafe"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

// USER_MODALS_INFO_0 — basic password policy
type userModalsInfo0 struct {
	Usrmod0_min_passwd_len    uint32
	Usrmod0_max_passwd_age    uint32
	Usrmod0_min_passwd_age    uint32
	Usrmod0_force_logoff      uint32
	Usrmod0_password_hist_len uint32
}

// USER_MODALS_INFO_3 — lockout policy
type userModalsInfo3 struct {
	Usrmod3_lockout_duration                uint32
	Usrmod3_lockout_observation_window      uint32
	Usrmod3_lockout_threshold               uint32
}

const (
	maxPasswordAgeNever = 0xFFFFFFFF // TIMEQ_FOREVER
	secondsPerDay       = 86400
	secondsPerMinute    = 60
)

var (
	procNetUserModalsGet = netapi32.NewProc("NetUserModalsGet")
)

func collectPasswordPolicy() (result reporter.PasswordPolicyInfo) {
	err := safeCollect("password_policy", func() error {
		// Level 0: basic password policy
		var buf0 uintptr
		ret, _, _ := procNetUserModalsGet.Call(
			0, // local computer
			0, // level 0
			uintptr(unsafe.Pointer(&buf0)),
		)
		if ret != 0 {
			return fmt.Errorf("NetUserModalsGet level 0 failed: %d", ret)
		}
		if buf0 == 0 {
			return fmt.Errorf("NetUserModalsGet returned nil buffer")
		}
		defer procNetApiBufferFree.Call(buf0)

		info0 := (*userModalsInfo0)(unsafe.Pointer(buf0))
		result.MinPasswordLength = int(info0.Usrmod0_min_passwd_len)
		result.PasswordHistoryCount = int(info0.Usrmod0_password_hist_len)

		if info0.Usrmod0_max_passwd_age == maxPasswordAgeNever {
			result.MaxPasswordAgeDays = 0 // 0 means never expires
		} else {
			result.MaxPasswordAgeDays = int(info0.Usrmod0_max_passwd_age / secondsPerDay)
		}

		result.MinPasswordAgeDays = int(info0.Usrmod0_min_passwd_age / secondsPerDay)

		// Level 3: lockout policy
		var buf3 uintptr
		ret, _, _ = procNetUserModalsGet.Call(
			0,
			3, // level 3
			uintptr(unsafe.Pointer(&buf3)),
		)
		if ret == 0 && buf3 != 0 {
			defer procNetApiBufferFree.Call(buf3)
			info3 := (*userModalsInfo3)(unsafe.Pointer(buf3))
			result.LockoutThreshold = int(info3.Usrmod3_lockout_threshold)
			result.LockoutDurationMinutes = int(info3.Usrmod3_lockout_duration / secondsPerMinute)
			result.LockoutObservationWindowMinutes = int(info3.Usrmod3_lockout_observation_window / secondsPerMinute)
		}

		// Password complexity is set by secedit/Group Policy and written to the LSA registry key.
		// Readable by SYSTEM; returns false (conservative) if the key is absent.
		result.ComplexityEnabled = registryCheckDWORD(
			`SYSTEM\CurrentControlSet\Control\Lsa`, "PasswordComplexity", 1)

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
