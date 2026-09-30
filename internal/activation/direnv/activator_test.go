package direnv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/activation"
)

// The direnv backend must be testable on its own, without any provider
// retrieval or vault access.
func TestActivatorRenderInstallRemove(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	a := &Activator{}
	if got := a.Name(); got != "direnv" {
		t.Fatalf("Name() = %q, want direnv", got)
	}

	cfg := activation.Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Team",
		FolderID:     "folder-1",
		Version:      "v3.0.0-test",
		ItemIDs:      []string{"i1"},
	}

	content, err := a.Render(cfg)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if _, err := os.Stat(".envrc"); !os.IsNotExist(err) {
		t.Fatal("Render() must not write .envrc")
	}
	if !strings.Contains(string(content), "bwenv export --project .") {
		t.Fatalf("rendered .envrc missing project export:\n%s", content)
	}

	if err := a.Install(cfg); err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	data, err := os.ReadFile(".envrc")
	if err != nil {
		t.Fatalf("read .envrc: %v", err)
	}
	if string(data) != string(content) {
		t.Fatal("installed .envrc differs from rendered content")
	}
	if _, err := os.Stat(".bwenv.toml"); err != nil {
		t.Fatalf("Install() must write .bwenv.toml: %v", err)
	}

	if err := a.Remove(); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if _, err := os.Stat(".envrc"); !os.IsNotExist(err) {
		t.Fatal("Remove() must delete .envrc")
	}
	if _, err := os.Stat(".bwenv.toml"); err != nil {
		t.Fatal("Remove() must leave .bwenv.toml to the caller")
	}
}

func TestActivatorIsRegistered(t *testing.T) {
	a, err := activation.Get("direnv")
	if err != nil {
		t.Fatalf("direnv backend is not registered: %v", err)
	}
	if a.Name() != "direnv" {
		t.Fatalf("Get(\"direnv\").Name() = %q", a.Name())
	}
}

func TestActivatorResolvesProject(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if err := Install(activation.Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Team",
		FolderID:     "folder-9",
		Version:      "v3.0.0-test",
	}); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	src, err := (&Activator{}).Resolve()
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if src.ProviderSlug != "bitwarden" || src.FolderName != "Team" || src.FolderID != "folder-9" {
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

func TestEnsureLoggingConfigPreservesUserSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DIRENV_CONFIG", dir)
	if err := EnsureLoggingConfig(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "direnv.toml")
	content := []byte("[global]\nstrict_env = true\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureLoggingConfig(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(content) {
		t.Fatalf("user configuration changed: %s, %v", got, err)
	}
}
