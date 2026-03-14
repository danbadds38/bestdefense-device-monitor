//go:build linux

package collector

import (
	"bufio"
	"os"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectLocalUsers() (result reporter.LocalUsersInfo) {
	err := safeCollect("users", func() error {
		// Build set of users in sudo or wheel group from /etc/group
		adminUsers := collectAdminGroupMembers()

		// Parse /etc/passwd
		f, err := os.Open("/etc/passwd")
		if err != nil {
			return err
		}
		defer f.Close()

		// Build UID→shell map and filter for real user accounts
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Split(line, ":")
			if len(fields) < 7 {
				continue
			}
			username := fields[0]
			// uid := fields[2]
			shell := fields[6]

			// Skip system accounts (UID < 1000 except root, or nologin shell)
			// We want root + human users with real shells
			if username != "root" && (strings.Contains(shell, "nologin") || strings.Contains(shell, "false")) {
				continue
			}

			user := reporter.LocalUser{
				Username:         username,
				IsLocal:          true,
				IsEnabled:        true,
				IsAdmin:          adminUsers[username] || username == "root",
				PasswordRequired: true,
			}

			result.Accounts = append(result.Accounts, user)
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// collectAdminGroupMembers returns usernames that are in the sudo or wheel group.
func collectAdminGroupMembers() map[string]bool {
	admins := make(map[string]bool)

	f, err := os.Open("/etc/group")
	if err != nil {
		return admins
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		groupName := fields[0]
		if groupName != "sudo" && groupName != "wheel" && groupName != "admin" {
			continue
		}
		for _, member := range strings.Split(fields[3], ",") {
			member = strings.TrimSpace(member)
			if member != "" {
				admins[member] = true
			}
		}
	}

	return admins
}
