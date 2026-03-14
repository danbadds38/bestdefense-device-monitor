//go:build windows

package collector

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"golang.org/x/sys/windows"
)

// USER_INFO_4 structure from the Windows API (simplified — we use USER_INFO_1 + admin check)
// We use NetUserEnum level 1 for basic info, then NetUserGetLocalGroups for admin check.

const (
	userInfoLevel1   = 1
	ufTempDuplicate  = 0x0001
	ufNormalAccount  = 0x0200
	ufDontExpirePW   = 0x10000
	ufAccountDisable = 0x0002
	ufPasswdNotreqd  = 0x0020
)

// userInfo1 mirrors the Win32 USER_INFO_1 structure.
type userInfo1 struct {
	Usri1_name         *uint16
	Usri1_password     *uint16
	Usri1_password_age uint32
	Usri1_priv         uint32
	Usri1_home_dir     *uint16
	Usri1_comment      *uint16
	Usri1_flags        uint32
	Usri1_script_path  *uint16
}

var (
	netapi32          = windows.NewLazySystemDLL("netapi32.dll")
	procNetUserEnum   = netapi32.NewProc("NetUserEnum")
	procNetApiBufferFree = netapi32.NewProc("NetApiBufferFree")
	procNetLocalGroupGetMembers = netapi32.NewProc("NetLocalGroupGetMembers")
)

func collectLocalUsers() (result reporter.LocalUsersInfo) {
	err := safeCollect("users", func() error {
		var (
			bufPtr    uintptr
			entriesRead uint32
			totalEntries uint32
			resumeHandle uint32
		)

		// NetUserEnum with level 1
		ret, _, _ := procNetUserEnum.Call(
			0, // local computer
			uintptr(userInfoLevel1),
			uintptr(ufNormalAccount),
			uintptr(unsafe.Pointer(&bufPtr)),
			0xFFFFFFFF, // maxLen — all entries
			uintptr(unsafe.Pointer(&entriesRead)),
			uintptr(unsafe.Pointer(&totalEntries)),
			uintptr(unsafe.Pointer(&resumeHandle)),
		)

		if ret != 0 && ret != 0x000000EA { // ERROR_MORE_DATA
			return fmt.Errorf("NetUserEnum failed: %d", ret)
		}
		if bufPtr == 0 {
			return nil
		}
		defer procNetApiBufferFree.Call(bufPtr)

		// Get the set of admin usernames
		admins := getAdminUsernames()

		stride := unsafe.Sizeof(userInfo1{})
		for i := uint32(0); i < entriesRead; i++ {
			ui := (*userInfo1)(unsafe.Pointer(bufPtr + uintptr(i)*stride))

			username := windows.UTF16PtrToString(ui.Usri1_name)
			if username == "" {
				continue
			}

			user := reporter.LocalUser{
				Username:             username,
				IsLocal:              true,
				IsEnabled:            ui.Usri1_flags&ufAccountDisable == 0,
				PasswordRequired:     ui.Usri1_flags&ufPasswdNotreqd == 0,
				PasswordNeverExpires: ui.Usri1_flags&ufDontExpirePW != 0,
				IsAdmin:              admins[strings.ToLower(username)],
			}

			// Password age in seconds → approximate last password set
			// (We don't collect this as PII, just flag if never expires)

			result.Accounts = append(result.Accounts, user)
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// getAdminUsernames returns a set of usernames that are members of the local Administrators group.
func getAdminUsernames() map[string]bool {
	admins := make(map[string]bool)

	// LOCALGROUP_MEMBERS_INFO_3 — member's domain\name
	type memberInfo3 struct {
		Lgrmi3_domainandname *uint16
	}

	var (
		bufPtr       uintptr
		entriesRead  uint32
		totalEntries uint32
		resumeHandle uintptr
	)

	groupName, _ := windows.UTF16PtrFromString("Administrators")

	ret, _, _ := procNetLocalGroupGetMembers.Call(
		0, // local computer
		uintptr(unsafe.Pointer(groupName)),
		3, // level 3 — domain\name
		uintptr(unsafe.Pointer(&bufPtr)),
		0xFFFFFFFF,
		uintptr(unsafe.Pointer(&entriesRead)),
		uintptr(unsafe.Pointer(&totalEntries)),
		uintptr(unsafe.Pointer(&resumeHandle)),
	)

	if ret != 0 || bufPtr == 0 {
		return admins
	}
	defer procNetApiBufferFree.Call(bufPtr)

	stride := unsafe.Sizeof(memberInfo3{})
	for i := uint32(0); i < entriesRead; i++ {
		mi := (*memberInfo3)(unsafe.Pointer(bufPtr + uintptr(i)*stride))
		full := windows.UTF16PtrToString(mi.Lgrmi3_domainandname)
		// Strip domain prefix if present (e.g., "DESKTOP\Administrator" → "administrator")
		if idx := strings.LastIndex(full, `\`); idx >= 0 {
			full = full[idx+1:]
		}
		admins[strings.ToLower(full)] = true
	}

	return admins
}

// windowsFileTimeToTime converts a Windows FILETIME (100-nanosecond intervals since 1601-01-01)
// to a time.Time. Returns nil for zero values.
func windowsFileTimeToTime(ft uint64) *time.Time {
	if ft == 0 {
		return nil
	}
	// Windows epoch: January 1, 1601
	// Unix epoch:    January 1, 1970
	// Difference: 11644473600 seconds
	const winToUnixEpochSecs = 11644473600
	unixSec := int64(ft/10000000) - winToUnixEpochSecs
	t := time.Unix(unixSec, 0).UTC()
	return &t
}
