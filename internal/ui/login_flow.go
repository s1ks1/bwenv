// Package ui — login flow for re-authenticating with the active provider.
// This file implements the "bwenv login" command which re-authenticates
// with the provider configured in .envrc, exports a fresh session token,
// and updates the .envrc — all without needing a full "bwenv init".
//
// This is the missing piece for day-to-day use: when a Bitwarden session
// expires, the user just runs "bwenv login" to unlock and resume, rather
// than going through the entire init wizard again.
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/envrc"
	"github.com/s1ks1/bwenv/internal/provider"
)

// RunLoginFlow is the interactive TTY handler for "bwenv login".
// It re-authenticates with the provider configured in .envrc, updates
// the BW_SESSION token in .envrc, verifies access to the configured
// folder, and runs direnv allow so that the next cd into the directory
// loads secrets automatically.
//
// Critical: after re-authenticating, this function writes the fresh
// BW_SESSION token back into the .envrc file. Without this step,
// direnv would re-fire with the stale token and fail immediately.
func RunLoginFlow(version string) error {
	PrintBanner(version)
	fmt.Println()

	// Step 1: Parse .envrc to find out which provider and folder are configured.
	providerSlug, folderName, err := envrc.ParseEnvrcConfig()
	if err != nil {
		PrintBoxError(
			E("❌", "[ERROR]")+" No bwenv configuration found",
			"",
			"  Could not find a valid .envrc in the current directory.",
			"  Run 'bwenv init' first to set up secrets for this project.",
		)
		return fmt.Errorf("no .envrc found: %w", err)
	}

	// Step 2: Look up and validate the provider.
	p, err := provider.Get(providerSlug)
	if err != nil {
		return fmt.Errorf("provider %q not found: %w", providerSlug, err)
	}

	if !p.IsAvailable() {
		PrintBoxError(
			E("❌", "[ERROR]")+fmt.Sprintf(" %s CLI (%s) is not installed", p.Name(), p.CLICommand()),
			"",
			fmt.Sprintf("  Install the '%s' CLI tool and try again.", p.CLICommand()),
		)
		return fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
	}

	// Show what we're doing.
	provName := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		Render(p.Name())
	folderStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		Render(folderName)

	PrintStep(1, 2, fmt.Sprintf("%s Authenticating with %s...", E("🔓", "[>]"), provName))
	fmt.Println()

	// Show current auth status for context.
	if p.IsAuthenticated() {
		PrintSuccess(E("🔓", "->") + " Already authenticated — session is valid")
		fmt.Println()
	} else {
		PrintInfo("Session expired — unlocking vault...")
		fmt.Println()
	}

	// Authenticate interactively (may prompt for master password / biometrics).
	session, err := p.Authenticate()
	if err != nil {
		PrintBoxError(
			E("❌", "[ERROR]")+" Authentication failed",
			"",
			"  "+err.Error(),
		)
		return fmt.Errorf("authentication failed: %w", err)
	}

	PrintSuccess(E("🔓", "->") + " Vault unlocked")

	// ── Critical: update .envrc with the fresh session token ──
	// Without this, direnv would re-fire with the stale BW_SESSION from .envrc
	// and "bwenv export" would fail with "session expired".
	if session != "" {
		if updateErr := envrc.UpdateSession(session); updateErr != nil {
			PrintWarning(fmt.Sprintf("Could not update .envrc with new session: %v", updateErr))
		} else {
			PrintSuccess("Updated session token in .envrc")
		}
	}

	fmt.Println()

	// Step 2: Verify we can access the configured folder.
	PrintStep(2, 2, fmt.Sprintf("%s Verifying access to %s...", E("📂", "[>]"), folderStyled))

	folders, err := p.ListFolders(session)
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	// Find the target folder.
	var targetFolder *provider.Folder
	for _, f := range folders {
		if f.Name == folderName {
			matched := f
			targetFolder = &matched
			break
		}
	}

	if targetFolder == nil {
		return fmt.Errorf("folder %q not found in %s", folderName, p.Name())
	}

	// Preview secrets to confirm they're accessible.
	varNames, err := envrc.PreviewSecrets(p, session, *targetFolder)
	if err != nil {
		PrintWarning(fmt.Sprintf("Could not verify secrets: %v", err))
	} else {
		PrintSuccess(fmt.Sprintf("%d variable(s) accessible", len(varNames)))
	}

	fmt.Println()

	// Allow direnv so next cd into the directory loads secrets automatically.
	// This must happen AFTER updating .envrc so direnv picks up the new token.
	if allowErr := envrc.AllowDirenv(); allowErr != nil {
		_ = allowErr // Non-fatal — direnv may not be installed.
	}

	// Show the success summary.
	var summaryLines []string
	summaryLines = append(summaryLines, E("✅", "[OK]")+" Session restored!")
	summaryLines = append(summaryLines, "")
	summaryLines = append(summaryLines, fmt.Sprintf("  Provider:   %s", p.Name()))
	summaryLines = append(summaryLines, fmt.Sprintf("  Folder:     %s", folderName))
	if len(varNames) > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("  Variables:  %d secret(s)", len(varNames)))
	}

	PrintBoxSuccess(summaryLines...)

	fmt.Println()

	// Hint about how to load secrets now.
	hint := lipgloss.NewStyle().Foreground(ColorMuted).
		Render("Secrets will load automatically on next cd into this directory.")
	fmt.Printf("  %s\n", hint)

	triggerHint := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).
		Render("  To load now: cd .")
	fmt.Println(triggerHint)
	fmt.Println()

	return nil
}
