package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/activation"
)

// stubBackend records approve/unapprove calls so activation can be exercised
// without direnv or any provider.
type stubBackend struct {
	approves   int
	unapproves int
}

func (s *stubBackend) Name() string                             { return "stub" }
func (s *stubBackend) Available() bool                          { return true }
func (s *stubBackend) Detect() activation.Status                { return activation.Status{Installed: true} }
func (s *stubBackend) Render(activation.Config) ([]byte, error) { return nil, nil }
func (s *stubBackend) Install(activation.Config) error          { return nil }
func (s *stubBackend) Remove() error                            { return nil }
func (s *stubBackend) Resolve() (activation.Source, error) {
	return activation.Source{ProviderSlug: "bitwarden", FolderName: "Team"}, nil
}
func (s *stubBackend) Approve() error   { s.approves++; return nil }
func (s *stubBackend) Unapprove() error { s.unapproves++; return nil }
func (s *stubBackend) Reload() error    { return nil }

const stubProjectConfig = "version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_name = \"Team\"\n[activation]\nmode = \"stub\"\n"

func TestActivateFromNestedDirIsIdempotent(t *testing.T) {
	backend := &stubBackend{}
	activation.Register(backend)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".bwenv.toml"), []byte(stubProjectConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, nested)

	for i := 0; i < 2; i++ {
		name, err := Activate()
		if err != nil {
			t.Fatalf("Activate() #%d error: %v", i+1, err)
		}
		if name != "stub" {
			t.Fatalf("Activate() backend = %q, want stub", name)
		}
	}
	if backend.approves != 2 {
		t.Fatalf("expected two idempotent approvals, got %d", backend.approves)
	}
}

func TestDeactivateClearsCachedVariables(t *testing.T) {
	backend := &stubBackend{}
	activation.Register(backend)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".bwenv.toml"), []byte(stubProjectConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".bwenv_vars"), []byte("API_KEY\nDB_URL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, nested)

	names, err := Deactivate()
	if err != nil {
		t.Fatalf("Deactivate() error: %v", err)
	}
	if len(names) != 3 { // API_KEY, DB_URL and the appended BW_SESSION
		t.Fatalf("Deactivate() names = %v, want 3", names)
	}
	if backend.unapproves != 1 {
		t.Fatalf("expected one unapproval, got %d", backend.unapproves)
	}
}

func TestActivateWithoutProjectFails(t *testing.T) {
	activation.Register(&stubBackend{})
	chdir(t, t.TempDir())

	if _, err := Activate(); err == nil {
		t.Fatal("expected an error when no project is found")
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
