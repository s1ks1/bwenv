package shell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/project"
)

func TestActivatorBasics(t *testing.T) {
	a := &Activator{}
	if a.Name() != "shell" {
		t.Fatalf("Name() = %q, want shell", a.Name())
	}
	if !a.Available() {
		t.Fatal("Available() should be true")
	}
	if !a.EmitsExports() {
		t.Fatal("shell backend must report EmitsExports")
	}

	content, err := a.Render(activation.Config{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(string(content), hookMarker) {
		t.Fatal("Render() missing the hook marker")
	}

	got, err := activation.Get("shell")
	if err != nil || got.Name() != "shell" {
		t.Fatalf("shell backend not registered: %v", err)
	}
}

func TestInstallIsIdempotentAndDetectable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("RC discovery is HOME-based and Unix-specific")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	a := &Activator{}

	if err := a.Install(activation.Config{}); err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if err := a.Install(activation.Config{}); err != nil {
		t.Fatalf("second Install() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	if n := strings.Count(string(data), hookMarker); n != 1 {
		t.Fatalf("hook installed %d times, want 1", n)
	}
	if st := a.Detect(); !st.Configured {
		t.Fatalf("Detect() = %+v, want Configured", st)
	}
}

func TestResolveReadsProjectConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	cfg := project.Config{
		Version:    project.ConfigVersion,
		Provider:   "bitwarden",
		Project:    project.Metadata{FolderID: "folder-1", FolderName: "Team", Items: []string{"i1"}},
		Activation: project.Activation{Mode: "shell"},
	}
	if err := project.Write(".bwenv.toml", cfg); err != nil {
		t.Fatalf("write project: %v", err)
	}

	src, err := (&Activator{}).Resolve()
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if src.ProviderSlug != "bitwarden" || src.FolderName != "Team" || src.FolderID != "folder-1" || len(src.ItemIDs) != 1 {
		t.Fatalf("Resolve() = %+v", src)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}
