package envrc

// PER-9 regression: automated coverage for user-facing handling of the
// provider-failure scenario on the fast export path. Additive-only test file;
// any real source bug found here is surfaced in the final report rather than
// patched (same convention as stability_test.go).

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// providerFailureInstallFakeCLI builds the shared fake CLI fixture (which
// honors BWENV_FAKE_SCENARIO and BWENV_FAKE_LOG) and puts it on PATH as bw.
func providerFailureInstallFakeCLI(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake CLI scenario is POSIX-oriented")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "bw")
	build := exec.Command("go", "build", "-o", binary, "../../tests/fixtures/fakecli")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake CLI: %v: %s", err, output)
	}
	t.Setenv("PATH", dir)
	t.Setenv("BWENV_FAKE_LOG", filepath.Join(dir, "calls.log"))
	return dir
}

// TestFastExportProviderFailureIsActionableAndSafe drives ExportWithFolderID
// against a provider that fails mid-export and asserts the failure is
// actionable, leak-free, and never syncs the vault.
func TestFastExportProviderFailureIsActionableAndSafe(t *testing.T) {
	const token = "super-secret-session-token-provider-xyz"
	dir := providerFailureInstallFakeCLI(t)
	t.Setenv("BWENV_FAKE_SCENARIO", "provider-error")
	t.Setenv("BW_SESSION", token)
	stabilityChdir(t, t.TempDir())

	stdout, stderr, err := stabilityCaptureOutput(t, func() error {
		return ExportWithFolderID("bitwarden", "Folder", "folder-id", nil)
	})
	if err == nil {
		t.Fatal("expected ExportWithFolderID to fail when the provider is unavailable")
	}

	// envrc.go wraps the provider error as
	//   "failed to get secrets from folder %q: %w"
	// where %w carries the CLI's own "provider unavailable" stderr. Assert on
	// those actual, stable substrings rather than inventing a message.
	if !strings.Contains(err.Error(), "failed to get secrets from folder") {
		t.Errorf("error should name the failing operation and folder, got: %v", err)
	}
	if !strings.Contains(err.Error(), "provider unavailable") {
		t.Errorf("error should surface the provider context, got: %v", err)
	}

	// printExportError renders a labeled box to stderr; the label is stable.
	if !strings.Contains(stderr, "Could not fetch secrets") {
		t.Errorf("stderr should carry the user-facing export error box, got: %q", stderr)
	}

	// The session token must never reach the user-facing error or stderr.
	for label, text := range map[string]string{"returned error": err.Error(), "stderr": stderr} {
		if strings.Contains(text, token) {
			t.Fatalf("session token leaked in %s: %s", label, text)
		}
	}

	// PER-9 AC: the export hot path must never sync the vault.
	if stdout != "" {
		t.Errorf("failed export should not emit eval-able stdout, got: %q", stdout)
	}
	calls, readErr := os.ReadFile(filepath.Join(dir, "calls.log"))
	if readErr != nil {
		t.Fatalf("read fake CLI log: %v", readErr)
	}
	if strings.Contains(string(calls), "sync") {
		t.Fatalf("export hot path must not sync the vault: %q", calls)
	}
	if !strings.Contains(string(calls), "list items") {
		t.Fatalf("expected the export to use the provider read path, log: %q", calls)
	}
}
