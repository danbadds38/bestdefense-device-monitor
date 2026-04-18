// Package scriptexec executes user-supplied scripts in a temporary file,
// enforces a configurable timeout, caps combined stdout+stderr at 1 MB,
// and cleans up the temp file before returning.
//
// Platform-specific interpreter selection is handled by the build-tagged
// files scriptexec_unix.go (bash/sh) and scriptexec_windows.go (PowerShell).
package scriptexec

import "time"

// Result holds the outcome of a single script execution.
type Result struct {
	// Stdout is the captured standard output (capped at maxOutputBytes/2).
	Stdout string
	// Stderr is the captured standard error (capped at maxOutputBytes/2).
	Stderr string
	// ExitCode is the process exit code, or -1 on timeout.
	ExitCode int
	// TimedOut is true when the process was killed because it exceeded the
	// configured timeout.
	TimedOut bool
}

// maxOutputBytes is the upper bound on the combined stdout+stderr bytes
// retained from any single execution.  Output beyond this limit is silently
// discarded at capture time so memory use is bounded.
const maxOutputBytes = 1 * 1024 * 1024 // 1 MB

// DefaultTimeout is used by callers that do not have an explicit per-task
// timeout from the server payload.
const DefaultTimeout = 5 * time.Minute

// limitedWriter wraps a byte accumulator and stops accepting bytes once cap
// is reached.  Excess bytes are silently discarded; Write always returns
// len(p), nil so callers (exec.Cmd) do not see spurious write errors.
type limitedWriter struct {
	buf []byte
	cap int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	remaining := lw.cap - len(lw.buf)
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		lw.buf = append(lw.buf, p...)
	}
	// Always report the full write to prevent exec from seeing an error.
	return len(p), nil
}

func (lw *limitedWriter) String() string {
	return string(lw.buf)
}
