package mise

import (
	"os"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/project"
)

func TestActivatorBasics(t *testing.T) {
	a := &Activator{}
	if a.Name() != "mise" {
		t.Fatalf("Name() = %q, want mise", a.Name())
	}

	content, err := a.Render(activation.Config{})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(string(content), "bwenv export --project .") {
		t.Fatalf("Render() must delegate to bwenv:\n%s", content)
	}

	got, err := activation.Get("mise")
	if err != nil || got.Name() != "mise" {
		t.Fatalf("mise backend not registered: %v", err)
	}
}

func TestInstallWritesDelegatingConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	a := &Activator{}
	if err := a.Install(activation.Config{}); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("Install() must write %s: %v", scriptPath, err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Install() must write %s: %v", configPath, err)
	}
	if !strings.Contains(string(data), "_.source") || !strings.Contains(string(data), scriptPath) {
		t.Fatalf("%s does not source the script:\n%s", configPath, data)
	}

	// Idempotent: a second install keeps a single bwenv block.
	if err := a.Install(activation.Config{}); err != nil {
		t.Fatalf("second Install() error: %v", err)
	}
	again, _ := os.ReadFile(configPath)
	if n := strings.Count(string(again), marker); n != 1 {
		t.Fatalf("expected one marker, got %d", n)
	}
}

func TestInstallRefusesToRewriteExistingMiseConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if err := os.WriteFile(configPath, []byte("[tools]\nnode = \"20\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := (&Activator{}).Install(activation.Config{}); err == nil {
		t.Fatal("expected Install() to refuse an existing mise.toml without the bwenv block")
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("the sourced script should still be written: %v", err)
	}
}

func TestResolveReadsProjectConfig(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if err := project.Write(".bwenv.toml", project.Config{
		Version:    project.ConfigVersion,
		Provider:   "bitwarden",
		Project:    project.Metadata{FolderName: "Team", FolderID: "folder-1"},
		Activation: project.Activation{Mode: "mise"},
	}); err != nil {
		t.Fatal(err)
	}

	src, err := (&Activator{}).Resolve()
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if src.ProviderSlug != "bitwarden" || src.FolderName != "Team" {
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
