// Package ui — logout flow for terminating active provider sessions.
// This file implements the "bwenv logout" command which locks all
// available provider vaults and clears session tokens for security.
package ui

import (
	"context"
	"os"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/provider"
)

// RunLogoutFlow locks all available provider vaults and reports the results.
// This is the main entry point for the "bwenv logout" command.
func RunLogoutFlow(version string) error {
	ctx := context.Background()
	allProviders := provider.Available()
	if len(allProviders) == 0 {
		PrintInfo("No provider CLI found")
		return nil
	}
	active := false
	for _, p := range allProviders {
		auth, err := provider.AsAuthenticator(p)
		if err != nil || !auth.IsAuthenticated(ctx) {
			continue
		}
		active = true
		locker, err := provider.AsLocker(p)
		if err == nil {
			err = locker.Lock(ctx)
		}
		if err != nil {
			PrintError("Could not lock "+p.Name(), err)
		} else {
			PrintSuccess(p.Name() + " vault locked")
		}
	}
	if !active {
		PrintInfo("No active sessions")
	}
	warnings := collectSessionEnvWarnings()
	if len(warnings) > 0 {
		names := make([]string, 0, len(warnings))
		for _, warning := range warnings {
			names = append(names, warning.name)
		}
		PrintWarning("Clear session variables from this shell: unset " + strings.Join(names, " "))
	}
	return nil
}

// sessionEnvWarning holds info about an env var that may contain a session token.
type sessionEnvWarning struct {
	name string // Environment variable name (e.g. "BW_SESSION").
	hint string // Explanation of what this variable is for.
}

// collectSessionEnvWarnings checks for environment variables that hold provider
// session tokens and returns warnings for any that are currently set.
func collectSessionEnvWarnings() []sessionEnvWarning {
	// Map of env var names to their descriptions.
	sessionVars := []struct {
		name string
		hint string
	}{
		{"BW_SESSION", "Bitwarden session token"},
		{"OP_SESSION", "1Password session token (legacy)"},
		{"OP_SERVICE_ACCOUNT_TOKEN", "1Password service account token"},
	}

	var warnings []sessionEnvWarning

	for _, sv := range sessionVars {
		// For OP_SESSION, check for any OP_SESSION_* variants.
		if sv.name == "OP_SESSION" {
			for _, env := range os.Environ() {
				if strings.HasPrefix(env, "OP_SESSION_") {
					parts := strings.SplitN(env, "=", 2)
					warnings = append(warnings, sessionEnvWarning{
						name: parts[0],
						hint: sv.hint,
					})
				}
			}
			continue
		}

		if val := os.Getenv(sv.name); val != "" {
			warnings = append(warnings, sessionEnvWarning{
				name: sv.name,
				hint: sv.hint,
			})
		}
	}

	return warnings
}
