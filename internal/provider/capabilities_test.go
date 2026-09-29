package provider

import (
	"context"
	"testing"
)

// minimalProvider implements only the core Provider contract.
type minimalProvider struct{}

func (minimalProvider) Name() string        { return "Minimal" }
func (minimalProvider) Slug() string        { return "minimal" }
func (minimalProvider) Description() string { return "core only" }
func (minimalProvider) CLICommand() string  { return "minimal" }
func (minimalProvider) IsAvailable() bool   { return true }

// fullProvider implements every optional capability on top of the core contract.
type fullProvider struct{ minimalProvider }

func (fullProvider) Authenticate(context.Context) (string, error)               { return "", nil }
func (fullProvider) AuthenticateNonInteractive(context.Context) (string, error) { return "", nil }
func (fullProvider) IsAuthenticated(context.Context) bool                       { return true }
func (fullProvider) ListFolders(context.Context, string) ([]Folder, error)      { return nil, nil }
func (fullProvider) ListItems(context.Context, string, Folder) ([]SecretItem, error) {
	return nil, nil
}
func (fullProvider) GetSecrets(context.Context, string, Folder) ([]Secret, error) { return nil, nil }
func (fullProvider) GetSecretsByItemIDs(context.Context, string, Folder, []string) ([]Secret, error) {
	return nil, nil
}
func (fullProvider) Lock(context.Context) error { return nil }

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

func TestCapabilityHelpersAcceptFullProvider(t *testing.T) {
	p := fullProvider{}

	if _, err := AsAuthenticator(p); err != nil {
		t.Fatalf("full provider should be an Authenticator: %v", err)
	}
	if _, err := AsFolderLister(p); err != nil {
		t.Fatalf("full provider should be a FolderLister: %v", err)
	}
	if _, err := AsSecretFetcher(p); err != nil {
		t.Fatalf("full provider should be a SecretFetcher: %v", err)
	}
	if _, err := AsLocker(p); err != nil {
		t.Fatalf("full provider should be a Locker: %v", err)
	}
}
