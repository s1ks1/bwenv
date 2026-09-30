// Package session manages provider session tokens for the CLI. Sessions are
// runtime state and never project configuration.
package session

import (
	"context"
	"fmt"

	"github.com/s1ks1/bwenv/v3/internal/provider"
)

// Reauthenticate authenticates with the named provider and returns a fresh
// session token. It is used by callers that need a valid session without
// exporting secrets (e.g. "bwenv allow" in TTY mode).
//
// It returns ("", nil) for providers that do not use session tokens (1Password).
func Reauthenticate(ctx context.Context, providerSlug string) (string, error) {
	p, err := provider.Get(providerSlug)
	if err != nil {
		return "", fmt.Errorf("provider %q not found: %w", providerSlug, err)
	}

	if !p.IsAvailable() {
		return "", fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
	}

	auth, err := provider.AsAuthenticator(p)
	if err != nil {
		return "", err
	}

	session, err := auth.Authenticate(ctx)
	if err != nil {
		return "", fmt.Errorf("authentication failed for %s: %w", p.Name(), err)
	}

	return session, nil
}
