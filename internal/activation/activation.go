// Package activation defines the replaceable activation boundary: how a
// project's secrets reach the shell. direnv is the stable default backend; the
// native shell hooks and the mise adapter are later experimental backends.
//
// A backend never retrieves secrets. It only installs and controls the
// activation artifacts, so the secret-retrieval path stays backend-agnostic.
package activation

import (
	"fmt"
	"sort"
	"strings"
)

// Config describes the project a backend must activate. It never contains
// secret values.
type Config struct {
	ProviderSlug string
	FolderName   string
	FolderID     string
	Session      string
	Version      string
	ItemIDs      []string
	ItemNames    []string
}

// Source is the provider folder a project reads secrets from. It is resolved by
// the activation backend from the project's artifacts.
type Source struct {
	ProviderSlug string
	FolderName   string
	FolderID     string
	ItemIDs      []string
}

// Status reports a backend's readiness for the current project.
type Status struct {
	Installed  bool   // backend tooling is present
	Configured bool   // the project is wired for this backend
	Detail     string // short human-readable detail
}

// Activator is the boundary every activation backend implements.
type Activator interface {
	// Name is the activation.mode value that selects this backend.
	Name() string

	// Available reports whether the backend's tooling is present.
	Available() bool

	// Detect reports backend readiness for the current project.
	Detect() Status

	// Render returns the activation artifact for cfg without writing it.
	Render(cfg Config) ([]byte, error)

	// Install writes the activation artifact (and any project metadata).
	Install(cfg Config) error

	// Remove deletes the activation artifact for the current project.
	Remove() error

	// Resolve reports the provider source the current project reads from.
	Resolve() (Source, error)

	// Approve trusts the activation artifact (direnv: allow).
	Approve() error

	// Unapprove revokes trust (direnv: deny).
	Unapprove() error

	// Reload asks the backend to reload the current environment.
	Reload() error
}

// registry holds every registered backend, keyed by its mode name.
var registry = map[string]Activator{}

// Register adds a backend to the global registry. Called from backend init().
func Register(a Activator) {
	registry[strings.ToLower(a.Name())] = a
}

// Get returns a backend by its mode name (case-insensitive).
func Get(mode string) (Activator, error) {
	a, ok := registry[strings.ToLower(mode)]
	if !ok {
		return nil, fmt.Errorf("unknown activation mode %q — available: %s", mode, availableNames())
	}
	return a, nil
}

// All returns every registered backend, sorted by name for deterministic output.
func All() []Activator {
	all := make([]Activator, 0, len(registry))
	for _, a := range registry {
		all = append(all, a)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name() < all[j].Name() })
	return all
}

func availableNames() string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
