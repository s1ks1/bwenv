package envrc

import (
	"os"
	"path/filepath"
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

	if !strings.Contains(text, "--provider bitwarden") {
		t.Error("expected .envrc to contain '--provider bitwarden'")
	}

	if !strings.Contains(text, "--folder 'Test Folder'") {
		t.Error("expected .envrc to contain '--folder' with folder name")
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

	if !strings.Contains(text, "--items") {
		t.Error("expected --items flag in .envrc")
	}

	if !strings.Contains(text, "item1,item2") {
		t.Error("expected item IDs after --items flag")
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

	provider, folder, itemIDs, err := ParseEnvrcConfig()
	if err != nil {
		t.Fatalf("ParseEnvrcConfig() returned error: %v", err)
	}

	if provider != "bitwarden" {
		t.Errorf("expected provider 'bitwarden', got %q", provider)
	}

	if folder != "My Secrets" {
		t.Errorf("expected folder 'My Secrets', got %q", folder)
	}

	if len(itemIDs) != 2 || itemIDs[0] != "Item A" || itemIDs[1] != "Item B" {
		t.Errorf("expected item IDs [Item A Item B], got %v", itemIDs)
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

// ── PreviewSecrets ──────────────────────────────────────────────────────────

type mockProvider struct {
	name              string
	secrets           []provider.Secret
	secretsByIDs      []provider.Secret
	listItemsResult   []provider.SecretItem
	getSecretsErr     error
	getSecretsByIDsErr error
}

func (m *mockProvider) Name() string                                             { return m.name }
func (m *mockProvider) Slug() string                                             { return "mock" }
func (m *mockProvider) Description() string                                      { return "Mock provider for testing" }
func (m *mockProvider) CLICommand() string                                       { return "mock" }
func (m *mockProvider) IsAvailable() bool                                        { return true }
func (m *mockProvider) IsAuthenticated() bool                                    { return true }
func (m *mockProvider) Authenticate() (string, error)                            { return "session", nil }
func (m *mockProvider) Lock() error                                              { return nil }
func (m *mockProvider) ListFolders(string) ([]provider.Folder, error)            { return nil, nil }
func (m *mockProvider) ListItems(string, provider.Folder) ([]provider.SecretItem, error) {
	return m.listItemsResult, nil
}
func (m *mockProvider) GetSecrets(string, provider.Folder) ([]provider.Secret, error) {
	return m.secrets, m.getSecretsErr
}
func (m *mockProvider) GetSecretsByItemIDs(string, []string) ([]provider.Secret, error) {
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

	names, err := PreviewSecretsByIDs(p, "session", []string{"item-1"})
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
