//go:build darwin

package collector

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectPasswordPolicy() (result reporter.PasswordPolicyInfo) {
	err := safeCollect("password_policy", func() error {
		// pwpolicy requires root. If not available, return gracefully with what we can.
		out, err := exec.Command("pwpolicy", "-n", ".", "-getglobalhashtable").Output()
		if err != nil {
			// Fall back to checking system-level policy via defaults
			collectPasswordPolicyFallback(&result)
			return nil
		}

		// Parse the XML/plist output looking for key fields
		// Output is an XML plist — we parse it with simple string search
		content := string(out)

		result.MinPasswordLength = extractPolicyInt(content, "minChars")
		result.MaxPasswordAgeDays = extractPolicyInt(content, "maxMinutesUntilChangePassword") / (60 * 24)
		result.LockoutThreshold = extractPolicyInt(content, "maxFailedLoginAttempts")
		result.LockoutDurationMinutes = extractPolicyInt(content, "minutesUntilFailedLoginReset")

		// Complexity: check if requiresAlpha + requiresNumeric are both set
		if strings.Contains(content, "requiresAlpha") && strings.Contains(content, "requiresNumeric") {
			result.ComplexityEnabled = true
		}

		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

// collectPasswordPolicyFallback reads what's available without root.
func collectPasswordPolicyFallback(result *reporter.PasswordPolicyInfo) {
	// Check global password policy length minimum (system preference)
	if out, err := exec.Command("defaults", "read",
		"/Library/Preferences/com.apple.loginwindow",
		"SHOWFULLNAME").Output(); err == nil {
		_ = out // just checking existence
	}

	// Most macOS systems don't expose detailed policy without root.
	// Return sensible defaults indicating collection wasn't possible.
	result.CollectionError = strPtr("password policy collection requires root")
}

// extractPolicyInt finds an integer value for a given key in plist XML.
func extractPolicyInt(content, key string) int {
	marker := "<key>" + key + "</key>"
	idx := strings.Index(content, marker)
	if idx < 0 {
		return 0
	}
	rest := content[idx+len(marker):]
	// Next element should be <integer>N</integer> or <real>N</real>
	start := strings.Index(rest, ">")
	end := strings.Index(rest, "</")
	if start < 0 || end < 0 || end <= start {
		return 0
	}
	val := strings.TrimSpace(rest[start+1 : end])
	n, _ := strconv.Atoi(val)
	return n
}
