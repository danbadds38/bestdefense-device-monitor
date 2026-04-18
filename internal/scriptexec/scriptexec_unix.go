//go:build !windows

package scriptexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Execute writes content to a temporary shell script, runs it under bash
// (falling back to sh if bash is not in PATH), enforces timeout, caps
// output at maxOutputBytes/2 per stream, and removes the temp file before
// returning.
//
// When dryRun is true the script is executed with the additional environment
// variable BESTDEFENSE_DRY_RUN=1 so scripts can alter their behaviour
// accordingly (e.g. print a diff instead of applying changes).
func Execute(content string, dryRun bool, timeout time.Duration) (Result, error) {
	f, err := os.CreateTemp("", "bdagent-*.sh")
	if err != nil {
		return Result{}, fmt.Errorf("scriptexec: create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return Result{}, fmt.Errorf("scriptexec: write script: %w", err)
	}
	if err := f.Close(); err != nil {
		return Result{}, fmt.Errorf("scriptexec: close temp file: %w", err)
	}
	if err := os.Chmod(f.Name(), 0700); err != nil {
		return Result{}, fmt.Errorf("scriptexec: chmod script: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	interpreter := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		interpreter = "sh"
	}

	cmd := exec.CommandContext(ctx, interpreter, f.Name())

	if dryRun {
		cmd.Env = append(os.Environ(), "BESTDEFENSE_DRY_RUN=1")
	}

	stdoutW := &limitedWriter{cap: maxOutputBytes / 2}
	stderrW := &limitedWriter{cap: maxOutputBytes / 2}
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	runErr := cmd.Run()

	var res Result
	res.Stdout = stdoutW.String()
	res.Stderr = stderrW.String()

	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res, nil
	}

	if runErr != nil {
		exitErr, ok := runErr.(*exec.ExitError)
		if !ok {
			return Result{}, fmt.Errorf("scriptexec: run: %w", runErr)
		}
		res.ExitCode = exitErr.ExitCode()
	}

	return res, nil
}
