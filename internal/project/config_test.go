package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantFolderID  string
		wantItemCount int
		wantErr       bool
	}{
		{
			name: "valid",
			content: `version = 1
provider = "bitwarden"

[project]
folder_id = "folder-123"
folder_name = "Production"
items = ["item-1", "item-2"]

[activation]
mode = "direnv"
`,
			wantFolderID:  "folder-123",
			wantItemCount: 2,
		},
		{name: "legacy config without folder id", content: "version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_name = \"Production\"\n[activation]\nmode = \"direnv\"\n"},
		{name: "unsupported version", content: "version = 2\n", wantErr: true},
		{name: "missing folder name", content: "version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_id = \"id\"\n[activation]\nmode = \"direnv\"\n", wantErr: true},
		{name: "missing activation mode", content: "version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_id = \"id\"\nfolder_name = \"Production\"\n[activation]\n", wantErr: true},
		{name: "unknown session field", content: "version = 1\nprovider = \"bitwarden\"\nsession = \"secret\"\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".bwenv.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatalf("write project config: %v", err)
			}
			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected invalid project config to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			if cfg.Project.FolderID != tt.wantFolderID || len(cfg.Project.Items) != tt.wantItemCount {
				t.Fatalf("unexpected project config: %+v", cfg)
			}
		})
	}
}

// --project . passes a directory; Load must resolve it to <dir>/.bwenv.toml
// instead of trying to read the directory itself.
func TestLoadAcceptsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".bwenv.toml"),
		[]byte("version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_id = \"id\"\nfolder_name = \"Production\"\n[activation]\nmode = \"direnv\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load(%q) returned error: %v", dir, err)
	}
	if cfg.Project.FolderID != "id" {
		t.Fatalf("unexpected project config: %+v", cfg)
	}
}

func TestWriteThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".bwenv.toml")
	want := Config{
		Version:    ConfigVersion,
		Provider:   "bitwarden",
		Project:    Metadata{FolderID: "folder-9", FolderName: "Team", Items: []string{"i1", "i2"}},
		Activation: Activation{Mode: "direnv"},
	}
	if err := Write(path, want); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got.Provider != want.Provider || got.Project.FolderName != want.Project.FolderName {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
	if len(got.Project.Items) != 2 {
		t.Fatalf("items lost in round trip: %+v", got.Project.Items)
	}
}
