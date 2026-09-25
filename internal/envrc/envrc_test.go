package envrc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/internal/provider"
)

const testVersion = "v0.0.0-test"

// ── Generate ─────────────────────────────────────────────────────────────────

func TestGenerateWithoutItems(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Test Folder",
		FolderID:     "folder-id-123",
		Session:      "session-token",
		Version:      testVersion,
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	content, err := os.ReadFile(".envrc")
	if err != nil {
		t.Fatalf("could not read generated .envrc: %v", err)
	}

	text := string(content)

	if !strings.Contains(text, "bwenv export") {
		t.Error("expected .envrc to contain 'bwenv export'")
	}

	if !strings.Contains(text, "bwenv export --project .") {
		t.Error("expected .envrc to load its canonical project config")
	}

	if strings.Contains(text, "--items") {
		t.Error("did not expect --items flag when no items configured")
	}

	if !strings.Contains(text, "BW_SESSION='session-token'") {
		t.Error("expected .envrc to contain BW_SESSION token")
	}

	if !strings.Contains(text, "# Provider: bitwarden | Folder: Test Folder") {
		t.Error("expected header comment with provider and folder")
	}
	projectConfig, err := os.ReadFile(".bwenv.toml")
	if err != nil {
		t.Fatalf("could not read generated .bwenv.toml: %v", err)
	}
	if strings.Contains(string(projectConfig), "session-token") {
		t.Fatal("project metadata must not contain the session token")
	}
	loadedProject, err := LoadProjectConfig(".bwenv.toml")
	if err != nil {
		t.Fatalf("generated project config is invalid: %v", err)
	}
	if loadedProject.Project.FolderID != "folder-id-123" {
		t.Errorf("expected project metadata folder ID folder-id-123, got %q", loadedProject.Project.FolderID)
	}
}

func TestGenerateWithItems(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := Generate(Config{
		ProviderSlug: "1password",
		FolderName:   "Dev",
		FolderID:     "vault-id",
		Version:      testVersion,
		ItemIDs:      []string{"item1", "item2"},
		ItemNames:    []string{"API Keys", "Database"},
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	content, err := os.ReadFile(".envrc")
	if err != nil {
		t.Fatalf("could not read generated .envrc: %v", err)
	}

	text := string(content)

	if !strings.Contains(text, "bwenv export --project .") {
		t.Error("expected .envrc to load its canonical project config")
	}
	projectConfig, err := os.ReadFile(".bwenv.toml")
	if err != nil {
		t.Fatalf("could not read generated .bwenv.toml: %v", err)
	}
	if !strings.Contains(string(projectConfig), "item1") || !strings.Contains(string(projectConfig), "item2") {
		t.Error("expected project metadata to contain selected item IDs")
	}

	if !strings.Contains(text, "# Provider: 1password | Folder: Dev | Items: API Keys, Database") {
		t.Error("expected header comment with provider, folder, and items")
	}
}

func TestGenerateWithoutSession(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := Generate(Config{
		ProviderSlug: "1password",
		FolderName:   "Production",
		FolderID:     "vault-id",
		Version:      testVersion,
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	content, err := os.ReadFile(".envrc")
	if err != nil {
		t.Fatalf("could not read generated .envrc: %v", err)
	}

	if strings.Contains(string(content), "BW_SESSION") {
		t.Error("did not expect BW_SESSION in .envrc for session-less provider")
	}
}

func TestGenerateWithoutFolderIDKeepsLegacyExport(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	if err := os.WriteFile(".bwenv.toml", []byte("stale metadata"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Legacy",
		Session:      "session-token",
		Version:      testVersion,
	}); err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	content, err := os.ReadFile(".envrc")
	if err != nil {
		t.Fatalf("could not read generated .envrc: %v", err)
	}
	text := string(content)
	if strings.Contains(text, "--folder-id") {
		t.Fatal("legacy config should not emit an empty --folder-id flag")
	}
	if _, err := os.Stat(".bwenv.toml"); !os.IsNotExist(err) {
		t.Fatalf("legacy config without a folder ID should not create .bwenv.toml, stat error: %v", err)
	}

	_, folder, folderID, _, err := ParseEnvrcConfigWithFolderID()
	if err != nil {
		t.Fatalf("ParseEnvrcConfigWithFolderID() returned error: %v", err)
	}
	if folder != "Legacy" || folderID != "" {
		t.Fatalf("expected legacy folder without ID, got folder=%q id=%q", folder, folderID)
	}
}

func TestGenerateFilePermissions(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Test",
		FolderID:     "id",
		Version:      testVersion,
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	info, err := os.Stat(".envrc")
	if err != nil {
		t.Fatalf("could not stat .envrc: %v", err)
	}

	if runtime.GOOS == "windows" {
		t.Skip("Windows does not preserve Unix permission bits")
	}

	const expectedPerm = os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("expected .envrc permissions %o, got %o", expectedPerm, info.Mode().Perm())
	}
}

// ── ParseEnvrcConfig ─────────────────────────────────────────────────────────

func TestParseEnvrcConfig(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "My Secrets",
		FolderID:     "folder-id",
		Session:      "tok-123",
		Version:      testVersion,
		ItemIDs:      []string{"id-1", "id-2"},
		ItemNames:    []string{"Item A", "Item B"},
	})
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	provider, folder, folderID, itemIDs, err := ParseEnvrcConfigWithFolderID()
	if err != nil {
		t.Fatalf("ParseEnvrcConfig() returned error: %v", err)
	}

	if provider != "bitwarden" {
		t.Errorf("expected provider 'bitwarden', got %q", provider)
	}

	if folder != "My Secrets" {
		t.Errorf("expected folder 'My Secrets', got %q", folder)
	}

	if folderID != "folder-id" {
		t.Errorf("expected folder ID 'folder-id', got %q", folderID)
	}

	if len(itemIDs) != 2 || itemIDs[0] != "id-1" || itemIDs[1] != "id-2" {
		t.Errorf("expected item IDs [id-1 id-2], got %v", itemIDs)
	}
}

func TestParseEnvrcConfigWithoutItems(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Test",
		FolderID:     "id",
		Session:      "tok",
		Version:      testVersion,
	})
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	_, _, itemIDs, err := ParseEnvrcConfig()
	if err != nil {
		t.Fatalf("ParseEnvrcConfig() returned error: %v", err)
	}

	if itemIDs != nil {
		t.Errorf("expected nil item IDs, got %v", itemIDs)
	}
}

func TestParseEnvrcConfigNoFile(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	_, _, _, err := ParseEnvrcConfig()
	if err == nil {
		t.Fatal("expected error when no .envrc exists")
	}

	if !strings.Contains(err.Error(), "no .envrc found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseEnvrcConfigNotBwenv(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	os.WriteFile(".envrc", []byte("export FOO=bar\n"), 0600)

	_, _, _, err := ParseEnvrcConfig()
	if err == nil {
		t.Fatal("expected error for non-bwenv .envrc")
	}

	if !strings.Contains(err.Error(), "not generated by bwenv") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoadProjectConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
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
		},
		{name: "unsupported version", content: "version = 2\n", wantErr: true},
		{name: "missing folder id", content: "version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_name = \"Production\"\n[activation]\nmode = \"direnv\"\n", wantErr: true},
		{name: "missing activation mode", content: "version = 1\nprovider = \"bitwarden\"\n[project]\nfolder_id = \"id\"\nfolder_name = \"Production\"\n[activation]\n", wantErr: true},
		{name: "unknown session field", content: "version = 1\nprovider = \"bitwarden\"\nsession = \"secret\"\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".bwenv.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatalf("write project config: %v", err)
			}
			cfg, err := LoadProjectConfig(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected invalid project config to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadProjectConfig() returned error: %v", err)
			}
			if cfg.Project.FolderID != "folder-123" || len(cfg.Project.Items) != 2 {
				t.Fatalf("unexpected project config: %+v", cfg)
			}
		})
	}
}

func TestParseEnvrcConfigRejectsInvalidCanonicalConfig(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.WriteFile(".bwenv.toml", []byte("version = 99"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".envrc", []byte("# Generated by bwenv\n# Provider: bitwarden | Folder: Legacy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := ParseEnvrcConfigWithFolderID(); err == nil {
		t.Fatal("expected invalid canonical metadata to fail rather than fall back")
	}
}

func TestRemoveDeletesProjectMetadata(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	for _, name := range []string{".envrc", ".bwenv.toml"} {
		if err := os.WriteFile(name, []byte("bwenv"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	removed, _, err := Remove()
	if err != nil {
		t.Fatalf("Remove() returned error: %v", err)
	}
	if !removed {
		t.Fatal("expected project files to be removed")
	}
	if _, err := os.Stat(".bwenv.toml"); !os.IsNotExist(err) {
		t.Fatalf("expected .bwenv.toml to be removed, stat error: %v", err)
	}
}

// ── PreviewSecrets ──────────────────────────────────────────────────────────

type mockProvider struct {
	name               string
	secrets            []provider.Secret
	secretsByIDs       []provider.Secret
	listItemsResult    []provider.SecretItem
	getSecretsErr      error
	getSecretsByIDsErr error
}

func (m *mockProvider) Name() string                                  { return m.name }
func (m *mockProvider) Slug() string                                  { return "mock" }
func (m *mockProvider) Description() string                           { return "Mock provider for testing" }
func (m *mockProvider) CLICommand() string                            { return "mock" }
func (m *mockProvider) IsAvailable() bool                             { return true }
func (m *mockProvider) IsAuthenticated() bool                         { return true }
func (m *mockProvider) Authenticate() (string, error)                 { return "session", nil }
func (m *mockProvider) AuthenticateNonInteractive() (string, error)   { return "session", nil }
func (m *mockProvider) Lock() error                                   { return nil }
func (m *mockProvider) ListFolders(string) ([]provider.Folder, error) { return nil, nil }
func (m *mockProvider) ListItems(string, provider.Folder) ([]provider.SecretItem, error) {
	return m.listItemsResult, nil
}
func (m *mockProvider) GetSecrets(string, provider.Folder) ([]provider.Secret, error) {
	return m.secrets, m.getSecretsErr
}
func (m *mockProvider) GetSecretsByItemIDs(string, provider.Folder, []string) ([]provider.Secret, error) {
	return m.secretsByIDs, m.getSecretsByIDsErr
}

func TestPreviewSecrets(t *testing.T) {
	p := &mockProvider{
		secrets: []provider.Secret{
			{Key: "API_KEY", Value: "sk-123"},
			{Key: "DB_HOST", Value: "localhost"},
		},
	}

	names, err := PreviewSecrets(p, "session", provider.Folder{Name: "Test", ID: "id"})
	if err != nil {
		t.Fatalf("PreviewSecrets() returned error: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	if names[0] != "API_KEY" || names[1] != "DB_HOST" {
		t.Errorf("expected [API_KEY DB_HOST], got %v", names)
	}
}

func TestPreviewSecretsReturnsKeysOnly(t *testing.T) {
	p := &mockProvider{
		secrets: []provider.Secret{
			{Key: "PASSWORD", Value: "super-secret-value"},
		},
	}

	names, err := PreviewSecrets(p, "session", provider.Folder{Name: "Test", ID: "id"})
	if err != nil {
		t.Fatalf("PreviewSecrets() returned error: %v", err)
	}

	for _, name := range names {
		if strings.Contains(name, "super-secret") {
			t.Error("PreviewSecrets should not return secret values")
		}
	}
}

func TestPreviewSecretsByIDs(t *testing.T) {
	p := &mockProvider{
		secretsByIDs: []provider.Secret{
			{Key: "API_KEY", Value: "sk-456"},
		},
	}

	names, err := PreviewSecretsByIDs(p, "session", provider.Folder{Name: "Test", ID: "id"}, []string{"item-1"})
	if err != nil {
		t.Fatalf("PreviewSecretsByIDs() returned error: %v", err)
	}

	if len(names) != 1 || names[0] != "API_KEY" {
		t.Errorf("expected [API_KEY], got %v", names)
	}
}

// ── sanitizeKey ─────────────────────────────────────────────────────────────

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"API_KEY", "API_KEY"},
		{"db-host", "db_host"},
		{"123start", "_123start"},
		{"my.var.name", "my_var_name"},
		{"Hello World!", "Hello_World_"},
		{"", "_EMPTY_KEY"},
		{"___", "___"},
		{"a", "a"},
		{"a-b-c", "a_b_c"},
	}

	for _, tt := range tests {
		result := sanitizeKey(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ── shellQuote ──────────────────────────────────────────────────────────────

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"it's", "'it'\\''s'"},
		{"simple text", "'simple text'"},
		{"", "''"},
		{"price is $5", "'price is $5'"},
	}

	for _, tt := range tests {
		result := shellQuote(tt.input)
		if result != tt.expected {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ── shellEscape ─────────────────────────────────────────────────────────────

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bitwarden", "bitwarden"},
		{"1password", "1password"},
		{"my-provider", "my-provider"},
		{"hello world!", "helloworld"},
		{"test.provider_v2", "test.provider_v2"},
		{"", ""},
	}

	for _, tt := range tests {
		result := shellEscape(tt.input)
		if result != tt.expected {
			t.Errorf("shellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ── shortenHomePath ─────────────────────────────────────────────────────────

func TestShortenHomePath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(home, ".zshrc"), "~/.zshrc"},
		{home, "~"},
		{"/tmp/somewhere", "/tmp/somewhere"},
		{"/nonexistent/path", "/nonexistent/path"},
	}

	for _, tt := range tests {
		result := shortenHomePath(tt.input)
		if result != tt.expected {
			t.Errorf("shortenHomePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ── UpdateSession ───────────────────────────────────────────────────────────

func TestUpdateSession(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Test",
		FolderID:     "id",
		Session:      "old-token",
		Version:      testVersion,
	})

	err := UpdateSession("new-token")
	if err != nil {
		t.Fatalf("UpdateSession() returned error: %v", err)
	}

	content, _ := os.ReadFile(".envrc")
	if !strings.Contains(string(content), "BW_SESSION='new-token'") {
		t.Error("expected updated BW_SESSION in .envrc")
	}

	if strings.Contains(string(content), "BW_SESSION='old-token'") {
		t.Error("old BW_SESSION should have been replaced")
	}
}

func TestUpdateSessionEmptyDoesNothing(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	Generate(Config{
		ProviderSlug: "bitwarden",
		FolderName:   "Test",
		FolderID:     "id",
		Session:      "token",
		Version:      testVersion,
	})

	err := UpdateSession("")
	if err != nil {
		t.Fatalf("UpdateSession() with empty string returned error: %v", err)
	}
}

func TestUpdateSessionNoFile(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	err := UpdateSession("new-token")
	if err == nil {
		t.Fatal("expected error when no .envrc exists")
	}
}

func TestRefreshSyncsOnlySupportedProvidersBeforeDirenvReload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test commands use POSIX scripts")
	}

	for _, tt := range []struct {
		provider string
		name     string
		session  string
		synced   bool
		wantLog  string
	}{
		{provider: "bitwarden", name: "Bitwarden", session: "test-session", synced: true, wantLog: "bw:list folders --session test-session\nbw:sync\ndirenv:reload\n"},
		{provider: "1password", name: "1Password", wantLog: "op:vault list --format=json\ndirenv:reload\n"},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			projectDir := t.TempDir()
			if err := os.Chdir(projectDir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(oldDir) })

			binDir := filepath.Join(projectDir, "bin")
			if err := os.Mkdir(binDir, 0700); err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(projectDir, "commands.log")
			t.Setenv("PATH", binDir)
			t.Setenv("BW_SESSION", tt.session)
			t.Setenv("BWENV_REFRESH_LOG", logPath)

			for _, name := range []string{"bw", "op", "direnv"} {
				script := fmt.Sprintf("#!/bin/sh\nprintf '%s:%%s\\n' \"$*\" >> \"$BWENV_REFRESH_LOG\"\ncase \"$*\" in\n  'list folders --session test-session'|'vault list --format=json') printf '[]' ;;\nesac\n", name)
				if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
			}

			content := fmt.Sprintf("# Generated by bwenv\n# Provider: %s | Folder: Test\neval \"$(bwenv export --provider %s --folder-id 'folder-id' --folder 'Test')\"\n", tt.provider, tt.provider)
			if err := os.WriteFile(".envrc", []byte(content), 0600); err != nil {
				t.Fatal(err)
			}

			name, synced, err := Refresh()
			if err != nil {
				t.Fatalf("Refresh() returned error: %v", err)
			}
			if name != tt.name || synced != tt.synced {
				t.Fatalf("Refresh() = (%q, %t), want (%q, %t)", name, synced, tt.name, tt.synced)
			}
			log, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(log) != tt.wantLog {
				t.Fatalf("command order = %q, want %q", log, tt.wantLog)
			}
		})
	}
}
