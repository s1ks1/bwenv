package provider

import (
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	p, err := Get("bitwarden")
	if err != nil {
		t.Fatalf("expected bitwarden to be registered, got error: %v", err)
	}
	if p.Name() != "Bitwarden" {
		t.Errorf("expected Name() = Bitwarden, got %q", p.Name())
	}
	if p.Slug() != "bitwarden" {
		t.Errorf("expected Slug() = bitwarden, got %q", p.Slug())
	}
}

func TestGetCaseInsensitive(t *testing.T) {
	p, err := Get("BitWarden")
	if err != nil {
		t.Fatalf("expected case-insensitive lookup to work, got error: %v", err)
	}
	if p.Slug() != "bitwarden" {
		t.Errorf("expected slug bitwarden, got %q", p.Slug())
	}
}

func TestGetUnknownProvider(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestAllReturnsSorted(t *testing.T) {
	all := All()
	if len(all) < 2 {
		t.Fatalf("expected at least 2 providers, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Name() > all[i].Name() {
			t.Errorf("providers not sorted by name: %s > %s",
				all[i-1].Name(), all[i].Name())
		}
	}
}

func TestAllContainsBitwarden(t *testing.T) {
	all := All()
	found := false
	for _, p := range all {
		if p.Slug() == "bitwarden" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected All() to include bitwarden")
	}
}

func TestAvailableSlugsString(t *testing.T) {
	slugs := availableSlugs()
	if slugs == "" {
		t.Fatal("expected non-empty slugs string")
	}
	if slugs != "1password, bitwarden" && slugs != "bitwarden, 1password" {
		t.Errorf("expected '1password, bitwarden' or 'bitwarden, 1password', got %q", slugs)
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "short"},
		{"", ""},
	}

	for _, tt := range tests {
		result := truncateOutput([]byte(tt.input))
		if result != tt.expected {
			t.Errorf("truncateOutput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTruncateOutputLong(t *testing.T) {
	long := string(make([]byte, 500))
	for i := range long {
		long = long[:i] + "x" + long[i+1:]
	}
	result := truncateOutput([]byte(long))
	if len(result) > 310 {
		t.Errorf("expected truncated output < 310 chars, got %d", len(result))
	}
}

func TestSecretItemStruct(t *testing.T) {
	item := SecretItem{ID: "id-1", Name: "My Item"}
	if item.ID != "id-1" {
		t.Errorf("expected ID 'id-1', got %q", item.ID)
	}
	if item.Name != "My Item" {
		t.Errorf("expected Name 'My Item', got %q", item.Name)
	}
}

func TestFolderStruct(t *testing.T) {
	f := Folder{ID: "folder-1", Name: "Production"}
	if f.ID != "folder-1" {
		t.Errorf("expected ID 'folder-1', got %q", f.ID)
	}
	if f.Name != "Production" {
		t.Errorf("expected Name 'Production', got %q", f.Name)
	}
}

func TestSecretStruct(t *testing.T) {
	s := Secret{Key: "API_KEY", Value: "sk-123"}
	if s.Key != "API_KEY" {
		t.Errorf("expected Key 'API_KEY', got %q", s.Key)
	}
	if s.Value != "sk-123" {
		t.Errorf("expected Value 'sk-123', got %q", s.Value)
	}
}

func TestOPVaultToFolder(t *testing.T) {
	v := opVault{ID: "vault-1", Name: "My Vault"}
	f := v.ToFolder()
	if f.ID != "vault-1" {
		t.Errorf("expected folder ID 'vault-1', got %q", f.ID)
	}
	if f.Name != "My Vault" {
		t.Errorf("expected folder Name 'My Vault', got %q", f.Name)
	}
}
