//go:build linux

package collector

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectPasswordPolicy() (result reporter.PasswordPolicyInfo) {
	err := safeCollect("password_policy", func() error {
		// /etc/login.defs — covers most distros
		parseLoginDefs(&result)

		// PAM pwquality for complexity/min-length (Debian: common-password, RHEL: system-auth)
		parsePamPasswordQuality(&result)

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

func parseLoginDefs(result *reporter.PasswordPolicyInfo) {
	f, err := os.Open("/etc/login.defs")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.Atoi(fields[1])
		switch fields[0] {
		case "PASS_MAX_DAYS":
			result.MaxPasswordAgeDays = val
		case "PASS_MIN_DAYS":
			result.MinPasswordAgeDays = val
		case "PASS_MIN_LEN":
			result.MinPasswordLength = val
		case "PASS_WARN_AGE":
			// Not in schema but useful context; skip
		}
	}
}

// parsePamPasswordQuality checks PAM config files for pwquality settings.
func parsePamPasswordQuality(result *reporter.PasswordPolicyInfo) {
	candidates := []string{
		"/etc/pam.d/common-password",  // Debian/Ubuntu
		"/etc/pam.d/system-auth",      // RHEL/CentOS/Fedora
		"/etc/pam.d/password-auth",    // RHEL alternate
		"/etc/security/pwquality.conf",
	}

	for _, path := range candidates {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") {
				continue
			}

			// PAM line: password requisite pam_pwquality.so minlen=12 ...
			// or pwquality.conf: minlen = 12
			if strings.Contains(line, "pam_pwquality") || strings.HasPrefix(path, "/etc/security/pwquality") {
				parts := strings.Fields(line)
				for _, part := range parts {
					k, v, ok := strings.Cut(part, "=")
					if !ok {
						continue
					}
					n, _ := strconv.Atoi(strings.TrimSpace(v))
					switch strings.TrimSpace(k) {
					case "minlen":
						if n > 0 {
							result.MinPasswordLength = n
						}
					case "minclass", "dcredit", "ucredit", "lcredit", "ocredit":
						// Any complexity requirements → complexity enabled
						result.ComplexityEnabled = true
					case "retry":
						if n > 0 {
							result.LockoutThreshold = n
						}
					}
				}
			}

			// faillock / pam_tally2 for lockout
			if strings.Contains(line, "pam_faillock") || strings.Contains(line, "pam_tally2") {
				parts := strings.Fields(line)
				for _, part := range parts {
					k, v, ok := strings.Cut(part, "=")
					if !ok {
						continue
					}
					n, _ := strconv.Atoi(strings.TrimSpace(v))
					switch strings.TrimSpace(k) {
					case "deny":
						result.LockoutThreshold = n
					case "unlock_time":
						result.LockoutDurationMinutes = n / 60
					case "fail_interval":
						result.LockoutObservationWindowMinutes = n / 60
					}
				}
			}
		}
		f.Close()
	}
}
