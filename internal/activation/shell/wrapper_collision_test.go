package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	runshell "github.com/s1ks1/bwenv/v3/internal/shell"
)

// TestWrapperInstallNotBlockedByHookMarker is the regression test for PER-42:
// wrapperMarker ("# bwenv shell integration") used to be a prefix of
// hookMarker ("# bwenv shell integration (experimental)"), so installing the
// native hook first made InstallWrapper falsely report "already present" and
// silently skip the wrapper.
func TestWrapperInstallNotBlockedByHookMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/bash")

	hookSnippet, err := Hook("bash")
	if err != nil {
		t.Fatalf("Hook(bash): %v", err)
	}
	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, []byte(hookSnippet), 0644); err != nil {
		t.Fatalf("write rc: %v", err)
	}

	modified, _, err := runshell.InstallWrapper()
	if err != nil {
		t.Fatalf("InstallWrapper: %v", err)
	}
	if !modified {
		t.Fatal("InstallWrapper skipped installation: hook marker false-positive (PER-42)")
	}

	content, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	got := string(content)
	if !strings.Contains(got, hookMarker) {
		t.Error("hook snippet lost after wrapper install")
	}
	if !strings.Contains(got, "bwenv() {") {
		t.Error("wrapper function missing after install")
	}
}

// TestHookInstallNotBlockedByWrapper guards the reverse direction (PER-42 AC):
// a wrapper must not satisfy the hook's detection, so the hook still installs.
func TestHookInstallNotBlockedByWrapper(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/bash")
	chdir(t, t.TempDir())

	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, nil, 0644); err != nil {
		t.Fatalf("write rc: %v", err)
	}
	if modified, _, err := runshell.InstallWrapper(); err != nil || !modified {
		t.Fatalf("InstallWrapper on empty rc = (%v, %v), want (true, nil)", modified, err)
	}

	cfg := activation.Config{ProviderSlug: "bitwarden", FolderName: "Test", FolderID: "folder-id"}
	if err := (&Activator{}).Install(cfg); err != nil {
		t.Fatalf("hook Install after wrapper: %v", err)
	}
	content, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if !strings.Contains(string(content), hookMarker) {
		t.Fatal("hook not installed on top of wrapper")
	}
}

// TestInstallWrapperIdempotent ensures a second install does not append a
// duplicate wrapper function.
func TestInstallWrapperIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/bash")

	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, nil, 0644); err != nil {
		t.Fatalf("write rc: %v", err)
	}
	if modified, _, err := runshell.InstallWrapper(); err != nil || !modified {
		t.Fatalf("first InstallWrapper = (%v, %v), want (true, nil)", modified, err)
	}
	if modified, _, err := runshell.InstallWrapper(); err != nil || modified {
		t.Fatalf("second InstallWrapper = (%v, %v), want (false, nil)", modified, err)
	}
	content, _ := os.ReadFile(rc)
	if n := strings.Count(string(content), "bwenv() {"); n != 1 {
		t.Errorf("wrapper function appears %d times, want 1", n)
	}
}

// TestInstallWrapperRecognizesLegacyInstall keeps pre-rename installs (which
// carry the old "# bwenv shell integration — enables" comment) from getting a
// second wrapper appended after the marker change.
func TestInstallWrapperRecognizesLegacyInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/bash")

	legacy := "\n# bwenv shell integration — enables seamless secret management\n" +
		"# Commands like allow/disallow/remove/login modify your shell environment directly.\n" +
		"bwenv() {\n  command bwenv \"$@\"\n}\n"
	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, []byte(legacy), 0644); err != nil {
		t.Fatalf("write rc: %v", err)
	}

	if modified, _, err := runshell.InstallWrapper(); err != nil || !modified {
		t.Fatalf("InstallWrapper on legacy install = (%v, %v), want notice upgrade", modified, err)
	}
}

func TestLegacyWrapperUpgradesLoginWithoutEval(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/bash")
	legacy := "# bwenv shell integration — enables seamless secret management\nbwenv() {\ncase \"${1:-}\" in\nallow|disallow|deny|remove|clean|export|load) eval \"$(command bwenv \"$@\")\" ;;\n*) command bwenv \"$@\" ;;\nesac\n}\n"
	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	modified, _, err := runshell.InstallWrapper()
	if err != nil || !modified {
		t.Fatalf("legacy upgrade: %v, %v", modified, err)
	}
	if modified, _, err := runshell.InstallWrapper(); err != nil || modified {
		t.Fatalf("upgrade was not idempotent: %v, %v", modified, err)
	}
	data, err := os.ReadFile(rc)
	if err != nil || !strings.Contains(string(data), "|login|auth|activate|deactivate|refresh)") {
		t.Fatalf("login missing from wrapper: %v", err)
	}
}
