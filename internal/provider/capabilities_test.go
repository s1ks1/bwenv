package provider

import "testing"

// minimalProvider implements only the core Provider contract.
type minimalProvider struct{}

func (minimalProvider) Name() string        { return "Minimal" }
func (minimalProvider) Slug() string        { return "minimal" }
func (minimalProvider) Description() string { return "core only" }
func (minimalProvider) CLICommand() string  { return "minimal" }
func (minimalProvider) IsAvailable() bool   { return true }

func TestCapabilityHelpersRejectMissingCapabilities(t *testing.T) {
	p := minimalProvider{}

	if _, err := AsAuthenticator(p); err == nil {
		t.Fatal("expected AsAuthenticator to fail for a core-only provider")
	}
	if _, err := AsFolderLister(p); err == nil {
		t.Fatal("expected AsFolderLister to fail for a core-only provider")
	}
	if _, err := AsSecretFetcher(p); err == nil {
		t.Fatal("expected AsSecretFetcher to fail for a core-only provider")
	}
	if _, err := AsLocker(p); err == nil {
		t.Fatal("expected AsLocker to fail for a core-only provider")
	}
}

func TestCapabilityHelpersAcceptConcreteProvider(t *testing.T) {
	p, err := Get("bitwarden")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AsAuthenticator(p); err != nil {
		t.Fatalf("Bitwarden should be an Authenticator: %v", err)
	}
	if _, err := AsFolderLister(p); err != nil {
		t.Fatalf("Bitwarden should be a FolderLister: %v", err)
	}
	if _, err := AsSecretFetcher(p); err != nil {
		t.Fatalf("Bitwarden should be a SecretFetcher: %v", err)
	}
	if _, err := AsLocker(p); err != nil {
		t.Fatalf("Bitwarden should be a Locker: %v", err)
	}
}
