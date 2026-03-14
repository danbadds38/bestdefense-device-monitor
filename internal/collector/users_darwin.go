//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectLocalUsers() (result reporter.LocalUsersInfo) {
	err := safeCollect("local_users", func() error {
		// List all users from Directory Services
		out, err := exec.Command("dscl", ".", "list", "/Users").Output()
		if err != nil {
			return err
		}

		for _, username := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			username = strings.TrimSpace(username)
			if username == "" || strings.HasPrefix(username, "_") {
				continue // skip service accounts
			}

			// Skip known system accounts
			switch username {
			case "nobody", "root", "daemon", "Guest":
				continue
			}

			// Get UID to filter out system accounts (UID < 500)
			uidOut, err := exec.Command("dscl", ".", "-read",
				"/Users/"+username, "UniqueID").Output()
			if err != nil {
				continue
			}
			uid := parseUniqueID(string(uidOut))
			if uid > 0 && uid < 500 {
				continue // system account
			}

			user := reporter.LocalUser{
				Username: username,
				IsLocal:  true,
				IsEnabled: true, // dscl doesn't easily expose disabled state
			}

			// Check if admin
			adminCheck := exec.Command("dseditgroup", "-o", "checkmember",
				"-m", username, "admin")
			if err := adminCheck.Run(); err == nil {
				user.IsAdmin = true
			}

			result.Accounts = append(result.Accounts, user)
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// parseUniqueID extracts the numeric UID from dscl output like "UniqueID: 501"
func parseUniqueID(output string) int {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "UniqueID:") {
			var uid int
			if _, err := strings.NewReader(strings.TrimPrefix(line, "UniqueID: ")).Read(nil); err == nil {
				// Parse the value after "UniqueID: "
				val := strings.TrimSpace(strings.TrimPrefix(line, "UniqueID:"))
				for _, ch := range val {
					if ch >= '0' && ch <= '9' {
						uid = uid*10 + int(ch-'0')
					} else {
						break
					}
				}
				return uid
			}
		}
	}
	return -1
}
