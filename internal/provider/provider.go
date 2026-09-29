// Package provider defines the interface for secret providers (Bitwarden, 1Password, etc.)
// and a registry to look them up by name. Each provider knows how to authenticate,
// list folders/vaults, and retrieve secrets as key-value pairs.
package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/process"
)

// Secret represents a single key-value pair retrieved from a provider.
type Secret struct {
	Key   string // Environment variable name (e.g. "DATABASE_URL")
	Value string // Secret value (e.g. "postgres://...")
}

// Folder represents a folder or vault that contains secrets.
type Folder struct {
	ID   string // Unique identifier from the provider
	Name string // Human-readable name shown in the UI
}

// SecretItem represents a single secret item/entry within a folder.
// Items contain custom fields that become environment variables.
type SecretItem struct {
	ID   string // Unique identifier from the provider
	Name string // Human-readable name shown in the UI
}

// Provider is the minimal contract every secret provider fulfils: identity and
// availability. Richer behaviour is opted into through the capability
// interfaces below, so a provider is never forced to implement features it does
// not have (e.g. folders, sync).
type Provider interface {
	// Name returns the display name of this provider (e.g. "Bitwarden").
	Name() string

	// Slug returns the short identifier used in CLI flags (e.g. "bitwarden").
	Slug() string

	// Description returns a one-line description of the provider.
	Description() string

	// CLICommand returns the name of the CLI binary this provider depends on (e.g. "bw").
	CLICommand() string

	// IsAvailable checks if the provider's CLI tool is installed and reachable.
	IsAvailable() bool
}

// Authenticator is implemented by providers that sign in and hold a session.
type Authenticator interface {
	// Authenticate unlocks or signs in to the provider's vault.
	// Returns a session token (or empty string if not applicable).
	Authenticate(ctx context.Context) (session string, err error)

	// AuthenticateNonInteractive returns the current session without prompting
	// or performing sync. Hot-path commands validate access through the
	// requested operation itself.
	AuthenticateNonInteractive(ctx context.Context) (session string, err error)

	// IsAuthenticated checks if the user currently has a valid session.
	IsAuthenticated(ctx context.Context) bool
}

// FolderLister is implemented by providers that expose folders/vaults and items.
type FolderLister interface {
	// ListFolders returns all folders/vaults available in the provider.
	ListFolders(ctx context.Context, session string) ([]Folder, error)

	// ListItems returns all secret items within the given folder.
	ListItems(ctx context.Context, session string, folder Folder) ([]SecretItem, error)
}

// SecretFetcher is implemented by providers that retrieve secret key-values.
type SecretFetcher interface {
	// GetSecrets retrieves all key-value secrets from the specified folder.
	GetSecrets(ctx context.Context, session string, folder Folder) ([]Secret, error)

	// GetSecretsByItemIDs retrieves secrets only from the specified items.
	GetSecretsByItemIDs(ctx context.Context, session string, folder Folder, itemIDs []string) ([]Secret, error)
}

// Locker is implemented by providers that can terminate a session.
type Locker interface {
	// Lock terminates the current session / locks the vault.
	Lock(ctx context.Context) error
}

// Syncer is implemented by providers with a separate local sync operation.
type Syncer interface {
	Sync(ctx context.Context) error
}

// registry holds all registered providers, keyed by their slug.
var registry = map[string]Provider{}

// Register adds a provider to the global registry.
// This is typically called from init() functions in each provider file.
func Register(p Provider) {
	registry[strings.ToLower(p.Slug())] = p
}

// Get returns a provider by its slug (case-insensitive).
// Returns an error if the provider is not found.
func Get(slug string) (Provider, error) {
	p, ok := registry[strings.ToLower(slug)]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q — available: %s", slug, availableSlugs())
	}
	return p, nil
}

// GetWithRunner returns an isolated provider instance for diagnostics and tests.
func GetWithRunner(slug string, runner process.Runner) (Provider, error) {
	p, err := Get(slug)
	if err != nil {
		return nil, err
	}
	cloneable, ok := p.(interface{ WithRunner(process.Runner) Provider })
	if !ok {
		return nil, fmt.Errorf("provider %q does not support an injected runner", slug)
	}
	return cloneable.WithRunner(runner), nil
}

// All returns a list of every registered provider, sorted by name for
// deterministic output (map iteration order is random in Go).
func All() []Provider {
	providers := make([]Provider, 0, len(registry))
	for _, p := range registry {
		providers = append(providers, p)
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Name() < providers[j].Name()
	})
	return providers
}

// Available returns only providers whose CLI tool is installed on this system,
// sorted by name for deterministic output.
func Available() []Provider {
	var available []Provider
	for _, p := range registry {
		if p.IsAvailable() {
			available = append(available, p)
		}
	}
	sort.Slice(available, func(i, j int) bool {
		return available[i].Name() < available[j].Name()
	})
	return available
}

// availableSlugs returns a comma-separated list of registered provider slugs.
// Used in error messages to show the user valid options.
func availableSlugs() string {
	slugs := make([]string, 0, len(registry))
	for slug := range registry {
		slugs = append(slugs, slug)
	}
	return strings.Join(slugs, ", ")
}
