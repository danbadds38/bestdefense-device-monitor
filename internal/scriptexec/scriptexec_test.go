package scriptexec_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/scriptexec"
)

// TestExecuteReturnsStdout verifies that standard output from the script is
// captured and returned in Result.Stdout.
func TestExecuteReturnsStdout(t *testing.T) {
	res, err := scriptexec.Execute("echo hello_world", false, 10*time.Second)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello_world") {
		t.Errorf("Stdout = %q, want it to contain %q", res.Stdout, "hello_world")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if res.TimedOut {
		t.Error("TimedOut should be false for a fast script")
	}
}

// TestExecuteReturnsExitCodeZero verifies exit code 0 for a succeeding script.
func TestExecuteReturnsExitCodeZero(t *testing.T) {
	res, err := scriptexec.Execute("exit 0", false, 10*time.Second)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

// TestExecuteReturnsNonZeroExitCode verifies that a non-zero exit is reflected
// in Result.ExitCode without returning a Go error.
func TestExecuteReturnsNonZeroExitCode(t *testing.T) {
	res, err := scriptexec.Execute("exit 42", false, 10*time.Second)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}
	if res.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", res.ExitCode)
	}
}

// TestExecuteCapturesStderr verifies that standard error output is captured
// in Result.Stderr.
func TestExecuteCapturesStderr(t *testing.T) {
	res, err := scriptexec.Execute("echo error_message >&2", false, 10*time.Second)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if !strings.Contains(res.Stderr, "error_message") {
		t.Errorf("Stderr = %q, want it to contain %q", res.Stderr, "error_message")
	}
}

// TestExecuteTimesOut verifies that a script running longer than the timeout
// is killed and Result.TimedOut is set to true.
func TestExecuteTimesOut(t *testing.T) {
	res, err := scriptexec.Execute("sleep 60", false, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}
	if !res.TimedOut {
		t.Error("TimedOut should be true when the process exceeds the timeout")
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 on timeout", res.ExitCode)
	}
}

// TestExecuteCapsOutput verifies that output beyond maxOutputBytes (1 MB) is
// silently truncated and Execute does not error.
func TestExecuteCapsOutput(t *testing.T) {
	// yes x | head -c 1100000 generates ~1.1 MB of output.
	res, err := scriptexec.Execute("yes x | head -c 1100000", false, 30*time.Second)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	const halfMB = 512 * 1024
	if len(res.Stdout) > halfMB+1024 { // allow a small OS-buffering slop
		t.Errorf("Stdout length %d exceeds cap (%d)", len(res.Stdout), halfMB)
	}
}

// TestExecuteDryRunSetsEnvVar verifies that when dryRun=true the script runs
// with BESTDEFENSE_DRY_RUN=1 in its environment.
func TestExecuteDryRunSetsEnvVar(t *testing.T) {
	res, err := scriptexec.Execute(`echo "DRY=${BESTDEFENSE_DRY_RUN}"`, true, 10*time.Second)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if !strings.Contains(res.Stdout, "DRY=1") {
		t.Errorf("Stdout = %q, want BESTDEFENSE_DRY_RUN=1 to be visible inside script", res.Stdout)
	}
}

// TestExecuteConcurrentIsolation verifies that concurrent executions do not
// cross-contaminate each other's output or temp files.
func TestExecuteConcurrentIsolation(t *testing.T) {
	scripts := []string{
		"echo label_alpha",
		"echo label_beta",
		"echo label_gamma",
		"echo label_delta",
	}

	type result struct {
		idx int
		res scriptexec.Result
		err error
	}

	ch := make(chan result, len(scripts))
	var wg sync.WaitGroup
	for i, s := range scripts {
		wg.Add(1)
		go func(idx int, script string) {
			defer wg.Done()
			res, err := scriptexec.Execute(script, false, 10*time.Second)
			ch <- result{idx: idx, res: res, err: err}
		}(i, s)
	}
	wg.Wait()
	close(ch)

	for r := range ch {
		if r.err != nil {
			t.Errorf("goroutine %d: Execute() error: %v", r.idx, r.err)
			continue
		}
		expected := []string{"label_alpha", "label_beta", "label_gamma", "label_delta"}[r.idx]
		if !strings.Contains(r.res.Stdout, expected) {
			t.Errorf("goroutine %d: Stdout = %q, want %q", r.idx, r.res.Stdout, expected)
		}
	}
}
