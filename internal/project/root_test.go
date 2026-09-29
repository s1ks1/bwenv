package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootWalksUpToProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".bwenv.toml"), []byte("version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(nested)
	if err != nil {
		t.Fatalf("FindRoot() error: %v", err)
	}
	if got != root {
		t.Fatalf("FindRoot(%q) = %q, want %q", nested, got, root)
	}
}

func TestFindRootRecognisesLegacyEnvrc(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".envrc"), []byte("eval \"$(bwenv export --project .)\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(root)
	if err != nil {
		t.Fatalf("FindRoot() error: %v", err)
	}
	if got != root {
		t.Fatalf("FindRoot(%q) = %q, want %q", root, got, root)
	}
}

func TestFindRootReturnsEmptyWhenNoProject(t *testing.T) {
	got, err := FindRoot(t.TempDir())
	if err != nil {
		t.Fatalf("FindRoot() error: %v", err)
	}
	if got != "" {
		t.Fatalf("FindRoot() = %q, want empty", got)
	}
}
