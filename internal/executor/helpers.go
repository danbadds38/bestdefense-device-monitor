package executor

import (
	"os/exec"
	"strings"
)

// runSequential runs each command in order, collecting trimmed output from each.
// Returns combined output and the first error encountered.
func runSequential(cmds [][]string) (string, error) {
	var lines []string
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if s := strings.TrimSpace(string(out)); s != "" {
			lines = append(lines, s)
		}
		if err != nil {
			return strings.Join(lines, "\n"), err
		}
	}
	return strings.Join(lines, "\n"), nil
}
