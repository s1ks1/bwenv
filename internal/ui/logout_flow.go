// Package ui — logout flow for terminating active provider sessions.
// This file implements the "bwenv logout" command which locks all
// available provider vaults and clears session tokens for security.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/provider"
)

// RunLogoutFlow locks all available provider vaults and reports the results.
// This is the main entry point for the "bwenv logout" command.
//
// The flow:
//  1. Display a banner with the version.
//  2. Iterate over all available providers.
//  3. For each provider that is authenticated, call Lock() to terminate the session.
//  4. Print a summary of which providers were locked.
//  5. Remind the user about environment variables that may still hold tokens.
func RunLogoutFlow(version string) error {
	PrintBanner(version)
	fmt.Println()

	allProviders := provider.Available()

	if len(allProviders) == 0 {
		PrintWarning("No provider CLI tools found — nothing to log out of")
		return nil
	}

	PrintStep(1, 1, E("🔒", "[>]")+" Locking vaults...")
	fmt.Println()

	// Track results for the summary.
	type lockResult struct {
		name    string
		wasAuth bool
		locked  bool
		err     error
	}

	var results []lockResult

	for _, p := range allProviders {
		wasAuthenticated := p.IsAuthenticated()

		if !wasAuthenticated {
			results = append(results, lockResult{
				name:    p.Name(),
				wasAuth: false,
				locked:  false,
			})
			continue
		}

		// Attempt to lock/sign out.
		err := p.Lock()
		results = append(results, lockResult{
			name:    p.Name(),
			wasAuth: true,
			locked:  err == nil,
			err:     err,
		})
	}

	// Print results for each provider.
	anyLocked := false
	anyErrors := false

	for _, r := range results {
		if !r.wasAuth {
			PrintInfoLine(r.name, "no active session")
			continue
		}

		if r.locked {
			PrintSuccess(fmt.Sprintf("%s vault locked", r.name))
			anyLocked = true
		} else {
			PrintError(fmt.Sprintf("Failed to lock %s", r.name), r.err)
			anyErrors = true
		}
	}

	fmt.Println()

	// Warn about environment variables that may still hold session tokens
	// in the current shell process. Locking the vault invalidates the token
	// server-side, but the variable lingers until the shell is restarted.
	envWarnings := collectSessionEnvWarnings()
	if len(envWarnings) > 0 {
		warnTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning).
			Render(E("⚠ ", "! ") + "Active session variables detected:")

		fmt.Printf("  %s\n\n", warnTitle)

		for _, w := range envWarnings {
			varName := lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true).
				Render(w.name)

			hint := lipgloss.NewStyle().
				Foreground(ColorMuted).
				Render(w.hint)

			fmt.Printf("    %s  %s\n", varName, hint)
		}

		fmt.Println()

		clearHint := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Render("  To clear these from your current shell, run:")

		fmt.Println(clearHint)

		for _, w := range envWarnings {
			cmd := lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Render(fmt.Sprintf("    unset %s", w.name))
			fmt.Println(cmd)
		}

		fmt.Println()
	}

	// Final summary.
	if anyLocked && !anyErrors {
		var summaryLines []string
		summaryLines = append(summaryLines, E("✅", "[OK]")+" All sessions terminated")
		summaryLines = append(summaryLines, "")
		summaryLines = append(summaryLines, "  Your vaults are now locked.")
		summaryLines = append(summaryLines, "  Run 'bwenv init' to start a new session.")
		PrintBoxSuccess(summaryLines...)
	} else if anyErrors {
		PrintWarning("Some providers could not be locked — see errors above")
	} else {
		PrintInfo("No active sessions found — nothing to lock")
	}

	fmt.Println()

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
