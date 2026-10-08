package provider_test

import (
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/provider"
	_ "github.com/s1ks1/bwenv/v3/internal/provider/all"
)

func TestGetRegisteredProviders(t *testing.T) {
	for _, slug := range []string{"bitwarden", "1password"} {
		p, err := provider.Get(slug)
		if err != nil {
			t.Fatalf("Get(%q): %v", slug, err)
		}
		if p.Slug() != slug {
			t.Fatalf("Get(%q).Slug() = %q", slug, p.Slug())
		}
	}
}

func TestGetCaseInsensitive(t *testing.T) {
	p, err := provider.Get("BitWarden")
	if err != nil {
		t.Fatalf("Get(\"BitWarden\") error: %v", err)
	}
	if p.Slug() != "bitwarden" {
		t.Fatalf("slug = %q, want bitwarden", p.Slug())
	}
}

func TestGetUnknownProviderListsAvailable(t *testing.T) {
	_, err := provider.Get("nonexistent")
	if err == nil {
		t.Fatal("expected an error for an unknown provider")
	}
	msg := err.Error()
	if !strings.Contains(msg, "bitwarden") || !strings.Contains(msg, "1password") {
		t.Fatalf("error should list available providers, got: %q", msg)
	}
}

func TestAllIsSortedAndComplete(t *testing.T) {
	all := provider.All()
	if len(all) < 2 {
		t.Fatalf("expected at least 2 providers, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Name() > all[i].Name() {
			t.Fatalf("providers not sorted by name: %s > %s", all[i-1].Name(), all[i].Name())
		}
	}
	slugs := make(map[string]bool, len(all))
	for _, p := range all {
		slugs[p.Slug()] = true
	}
	if !slugs["bitwarden"] || !slugs["1password"] {
		t.Fatalf("All() is missing a built-in provider: %v", slugs)
	}
}

func TestModelStructs(t *testing.T) {
	if s := (provider.Secret{Key: "K", Value: "V"}); s.Key != "K" || s.Value != "V" {
		t.Fatal("Secret fields not preserved")
	}
	if f := (provider.Folder{ID: "f1", Name: "Prod"}); f.ID != "f1" || f.Name != "Prod" {
		t.Fatal("Folder fields not preserved")
	}
	if it := (provider.SecretItem{ID: "i1", Name: "Item"}); it.ID != "i1" || it.Name != "Item" {
		t.Fatal("SecretItem fields not preserved")
	}
}
