// Package activation defines the replaceable activation boundary: how a
// project's secrets reach the shell. Native shell hooks are the v3 default for
// new projects; direnv remains the legacy fallback and mise is optional.
//
// A backend never retrieves secrets. It only installs and controls the
// activation artifacts, so the secret-retrieval path stays backend-agnostic.
package activation

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/project"
)

// Config describes the project a backend must activate. It never contains
// secret values.
type Config struct {
	ProviderSlug string
	FolderName   string
	FolderID     string
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

// Emitter is implemented by backends that activate by printing shell export
// statements for the current shell (native shell hooks), instead of delegating
// to an external tool such as direnv.
type Emitter interface {
	EmitsExports() bool
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

// ForProject selects the configured activation backend. Projects without
// canonical metadata are treated as legacy direnv projects.
func ForProject() (Activator, error) {
	cfg, err := project.Load(".bwenv.toml")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return Get("direnv")
	}
	return Get(cfg.Activation.Mode)
}

// WriteProject persists the provider references shared by every activation
// backend. Session values are deliberately absent from this project model.
func WriteProject(cfg Config, mode string) error {
	return project.Write(".bwenv.toml", project.Config{
		Version:  project.ConfigVersion,
		Provider: cfg.ProviderSlug,
		Project: project.Metadata{
			FolderID:   cfg.FolderID,
			FolderName: cfg.FolderName,
			Items:      cfg.ItemIDs,
		},
		Activation: project.Activation{Mode: mode},
	})
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
