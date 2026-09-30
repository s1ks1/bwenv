package process

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/s1ks1/bwenv/v3/internal/diagnostics"
)

// TestExecRunnerHelperProcess is not a real test. The runner tests re-execute
// the test binary with -test.run pointing here to produce controlled output
// without depending on any external command.
func TestExecRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if msg := os.Getenv("HELPER_STDOUT"); msg != "" {
		_, _ = os.Stdout.WriteString(msg)
	}
	if msg := os.Getenv("HELPER_STDERR"); msg != "" {
		_, _ = os.Stderr.WriteString(msg)
	}
	if d := os.Getenv("HELPER_SLEEP"); d != "" {
		if dur, err := time.ParseDuration(d); err == nil {
			time.Sleep(dur)
		}
	}
	os.Exit(0)
}

func helperInvocation(t *testing.T, extraEnv ...string) (string, []string, IO) {
	t.Helper()
	env := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	env = append(env, extraEnv...)
	return os.Args[0], []string{"-test.run=TestExecRunnerHelperProcess"}, IO{Env: env}
}

func TestExecRunnerCapturesStdoutAndStderr(t *testing.T) {
	name, args, streams := helperInvocation(t, "HELPER_STDOUT=hello", "HELPER_STDERR=oops")

	result, err := (ExecRunner{}).Run(context.Background(), name, args, streams)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if string(result.Stdout) != "hello" {
		t.Fatalf("stdout = %q, want %q", result.Stdout, "hello")
	}
	if string(result.Stderr) != "oops" {
		t.Fatalf("stderr = %q, want %q", result.Stderr, "oops")
	}
}

func TestExecRunnerCountsProcessesInRecorder(t *testing.T) {
	recorder := diagnostics.NewRecorder()
	name, args, streams := helperInvocation(t, "HELPER_STDOUT=ok")

	if _, err := (ExecRunner{Recorder: recorder}).Run(context.Background(), name, args, streams); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	_, processes := recorder.Snapshot()
	if len(processes) != 1 || processes[0].Count != 1 {
		t.Fatalf("process was not counted exactly once: %+v", processes)
	}
}

func TestExecRunnerTimeoutKillsLongProcess(t *testing.T) {
	name, args, streams := helperInvocation(t, "HELPER_SLEEP=2s")

	start := time.Now()
	_, err := (ExecRunner{Timeout: 50 * time.Millisecond}).Run(context.Background(), name, args, streams)
	if err == nil {
		t.Fatal("expected a timeout error for a process that outlives the runner timeout")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("timeout did not stop the process promptly: %v", elapsed)
	}
}

func TestExecRunnerHonoursCancelledContext(t *testing.T) {
	name, args, streams := helperInvocation(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := (ExecRunner{}).Run(ctx, name, args, streams); err == nil {
		t.Fatal("expected an error for an already-cancelled context")
	}
}

func TestExecRunnerBoundsInheritedOutputPipes(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("requires a POSIX shell")
	}
	start := time.Now()
	_, err = (ExecRunner{Timeout: 50 * time.Millisecond}).Run(context.Background(), sh, []string{"-c", "sleep 2 & wait"}, IO{})
	if err == nil {
		t.Fatal("expected cancellation")
	}
	if elapsed := time.Since(start); elapsed > 1800*time.Millisecond {
		t.Fatalf("child-held pipes delayed cancellation: %v", elapsed)
	}
}
