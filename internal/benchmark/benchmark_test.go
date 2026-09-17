package benchmark

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/s1ks1/bwenv/internal/diagnostics"
	"github.com/s1ks1/bwenv/internal/process"
)

func installFakeCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	binary := filepath.Join(dir, "bw"+ext)
	cmd := exec.Command("go", "build", "-o", binary, "../../tests/fixtures/fakecli")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake CLI: %v: %s", err, output)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("BW_SESSION", "fake-session")
	t.Setenv("BWENV_FAKE_LOG", filepath.Join(dir, "calls.log"))
	return dir
}

func TestBenchmarkCountsCurrentBitwardenPathWithoutSecrets(t *testing.T) {
	dir := installFakeCLI(t)
	report, err := Benchmark("bitwarden", "Fixture", "folder-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := report.Print(&output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "fake-secret-value") {
		t.Fatal("benchmark leaked a secret value")
	}
	if report.Variables != 2 || len(report.Processes) != 1 || report.Processes[0].Count != 1 {
		t.Fatalf("unexpected baseline: variables=%d processes=%v", report.Variables, report.Processes)
	}
	calls, err := os.ReadFile(filepath.Join(dir, "calls.log"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(calls)), "\n"); len(lines) != 1 {
		t.Fatalf("expected one real CLI invocation, got %q", calls)
	}
	if strings.Contains(string(calls), "sync") {
		t.Fatalf("hot path must not sync the vault: %q", calls)
	}
}

func TestBenchmarkExpiredSessionRedactsProviderOutput(t *testing.T) {
	installFakeCLI(t)
	t.Setenv("BWENV_FAKE_SCENARIO", "expired")
	_, err := Benchmark("bitwarden", "Fixture", "folder-1", nil)
	if err == nil || !strings.Contains(err.Error(), "bwenv login") {
		t.Fatalf("expected actionable auth error, got %v", err)
	}
}

func TestBenchmarkSelectedItemsCount(t *testing.T) {
	installFakeCLI(t)
	report, err := Benchmark("bitwarden", "Fixture", "folder-1", []string{"item-1", "item-2"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Variables != 2 || report.Processes[0].Count != 1 {
		t.Fatalf("expected two selected items and one process, got %+v", report)
	}
}

func TestBenchmarkOnePassword(t *testing.T) {
	dir := installFakeCLI(t)
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	binary, err := os.ReadFile(filepath.Join(dir, "bw"+ext))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "op"+ext), binary, 0700); err != nil {
		t.Fatal(err)
	}
	report, err := Benchmark("1password", "Fixture", "vault-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Variables != 1 || report.Processes[0].Count != 2 {
		t.Fatalf("unexpected 1Password baseline: %+v", report)
	}
}

func TestBenchmarkMalformedProviderOutputIsGeneric(t *testing.T) {
	installFakeCLI(t)
	t.Setenv("BWENV_FAKE_SCENARIO", "malformed")
	_, err := Benchmark("bitwarden", "Fixture", "folder-1", nil)
	if err == nil || strings.Contains(err.Error(), "not-json") {
		t.Fatalf("malformed provider payload leaked: %v", err)
	}
}

func TestRunnerTimeoutCountsAttempt(t *testing.T) {
	installFakeCLI(t)
	t.Setenv("BWENV_FAKE_DELAY", "100ms")
	recorder := diagnostics.NewRecorder()
	result, err := (process.ExecRunner{Recorder: recorder, Timeout: 10 * time.Millisecond}).Run(context.Background(), "bw", []string{"list", "folders"}, process.IO{})
	if err == nil || len(result.Stdout) != 0 {
		t.Fatalf("expected timed-out process, got result=%q err=%v", result.Stdout, err)
	}
	_, processes := recorder.Snapshot()
	if len(processes) != 1 || processes[0].Count != 1 {
		t.Fatalf("timed-out process was not counted: %v", processes)
	}
}
