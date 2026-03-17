// Package provider defines the interface for secret providers (Bitwarden, 1Password, etc.)
// and a registry to look them up by name. Each provider knows how to authenticate,
// list folders/vaults, and retrieve secrets as key-value pairs.
package provider

import (
	"fmt"
	"sort"
	"strings"
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

// Provider is the interface that all secret providers must implement.
// Each provider wraps a CLI tool (bw, op, etc.) and exposes a uniform API.
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

	// IsAuthenticated checks if the user is currently logged in / has a valid session.
	IsAuthenticated() bool

	// Authenticate unlocks or signs in to the provider's vault.
	// Returns a session token (or empty string if not applicable).
	Authenticate() (session string, err error)

	// ListFolders returns all folders/vaults available in the provider.
	// The session parameter may be needed for providers like Bitwarden.
	ListFolders(session string) ([]Folder, error)

	// GetSecrets retrieves all key-value secrets from the specified folder.
	// The session parameter may be needed for providers like Bitwarden.
	GetSecrets(session string, folder Folder) ([]Secret, error)

	// Lock terminates the current session / locks the vault.
	// For Bitwarden this runs "bw lock", for 1Password "op signout".
	// Returns nil if the provider has no active session or locking is not applicable.
	Lock() error
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
