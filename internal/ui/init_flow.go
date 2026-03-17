// Package ui — init flow orchestrator.
// This file ties together all the TUI components (provider picker, folder picker,
// authentication, and .envrc generation) into a single interactive flow.
// It is the main entry point for the "bwenv init" command.
//
// The flow now also:
//   - Previews which variables will be loaded (showing names, not values)
//   - Automatically runs "direnv allow" so the user doesn't see the scary
//     "direnv: error .envrc is blocked" message
//   - Shows a beautiful final summary with emojis (configurable)
//   - Respects user config for direnv output silencing
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/config"
	"github.com/s1ks1/bwenv/internal/envrc"
	"github.com/s1ks1/bwenv/internal/provider"
)

// RunInitFlow executes the full interactive initialization process:
//  1. Display a welcome banner with version info.
//  2. Let the user pick a secret provider (Bitwarden, 1Password, etc.).
//  3. Authenticate with the chosen provider (unlock vault / sign in).
//  4. Fetch and display the list of folders/vaults from the provider.
//  5. Let the user pick a folder to load secrets from.
//  6. Preview which environment variables will be loaded.
//  7. Generate a .envrc file in the current directory.
//  8. Automatically run "direnv allow" to approve the .envrc.
//
// Returns an error if any step fails or if the user cancels.
func RunInitFlow(version string) error {
	// Total number of steps shown to the user (for the [N/M] progress indicator).
	const totalSteps = 6

	// -- Step 0: Show the welcome banner --
	PrintBanner(version)

	// -- Step 1: Gather all registered providers --
	allProviders := provider.All()
	if len(allProviders) == 0 {
		return fmt.Errorf("no secret providers are registered — this is a bug in bwenv")
	}

	// Check if at least one provider's CLI is available on this system.
	available := provider.Available()
	if len(available) == 0 {
		printNoProvidersHelp(allProviders)
		return fmt.Errorf("no supported password manager CLI tools found on this system")
	}

	// -- Step 2: Provider selection --
	// If only one provider is available, skip the picker and use it directly.
	var chosenProvider provider.Provider

	if len(available) == 1 {
		chosenProvider = available[0]
		PrintStep(1, totalSteps, fmt.Sprintf("%s Using %s (only available provider)", E("🔑", "[>]"), formatProviderName(chosenProvider.Name())))
		fmt.Println()
	} else {
		// Launch the interactive provider picker TUI.
		PrintStep(1, totalSteps, E("🔑", "[>]")+" Select a secret provider")
		fmt.Println()

		pickerModel := NewProviderPicker(allProviders)
		program := tea.NewProgram(pickerModel)

		finalModel, err := program.Run()
		if err != nil {
			return fmt.Errorf("provider picker failed: %w", err)
		}

		result := finalModel.(ProviderPickerModel)
		if result.Cancelled() {
			printCancelled()
			os.Exit(0)
		}

		chosenProvider = result.Chosen()
		if chosenProvider == nil {
			return fmt.Errorf("no provider was selected")
		}

		fmt.Println()
		PrintSuccess(fmt.Sprintf("Selected: %s", chosenProvider.Name()))
	}

	// -- Step 3: Authenticate with the provider --
	PrintStep(2, totalSteps, E("🔓", "[>]")+" Authenticating with "+chosenProvider.Name()+"...")
	fmt.Println()

	session, err := chosenProvider.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	PrintSuccess(E("🔓", "->") + " Vault unlocked")
	fmt.Println()

	// -- Step 4: Fetch the folder list --
	PrintStep(3, totalSteps, E("📂", "[>]")+" Fetching folders from "+chosenProvider.Name()+"...")

	folders, err := chosenProvider.ListFolders(session)
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	if len(folders) == 0 {
		return fmt.Errorf("no folders found in your %s account — create one first", chosenProvider.Name())
	}

	PrintSuccess(fmt.Sprintf("Found %d folder(s)", len(folders)))
	fmt.Println()

	// -- Step 5: Folder selection via interactive TUI --
	PrintStep(4, totalSteps, E("📁", "[>]")+" Pick a folder to load secrets from")
	fmt.Println()

	folderModel := NewFolderPicker(folders, chosenProvider.Name())
	folderProgram := tea.NewProgram(folderModel)

	finalFolderModel, err := folderProgram.Run()
	if err != nil {
		return fmt.Errorf("folder picker failed: %w", err)
	}

	folderResult := finalFolderModel.(FolderPickerModel)
	if folderResult.Cancelled() {
		printCancelled()
		os.Exit(0)
	}

	chosenFolder := folderResult.Chosen()
	if chosenFolder == nil {
		return fmt.Errorf("no folder was selected")
	}

	fmt.Println()
	PrintSuccess(fmt.Sprintf("Selected: %s", chosenFolder.Name))
	fmt.Println()

	// -- Step 6: Preview secrets (show variable names, not values) --
	PrintStep(5, totalSteps, E("🔍", "[>]")+" Scanning secrets in folder...")

	varNames, err := envrc.PreviewSecrets(chosenProvider, session, *chosenFolder)
	if err != nil {
		// Non-fatal — we can still generate the .envrc, but warn the user.
		PrintWarning(fmt.Sprintf("Could not preview secrets: %v", err))
		PrintInfo("The .envrc will still be generated — secrets will load when direnv runs it.")
		fmt.Println()
	} else if len(varNames) == 0 {
		PrintWarning("No secrets found in this folder")
		PrintInfo("Make sure your vault items have custom fields with names and values.")
		fmt.Println()
	} else {
		// Show a nice preview of which variables will be loaded.
		printVariablePreview(varNames)
		fmt.Println()
	}

	// -- Step 7: Generate the .envrc file --
	PrintStep(6, totalSteps, E("📝", "[>]")+" Generating .envrc...")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	envrcPath := filepath.Join(cwd, ".envrc")

	// Check if .envrc already exists and warn the user.
	if _, statErr := os.Stat(envrcPath); statErr == nil {
		PrintWarning("Existing .envrc will be overwritten")
	}

	// Build and write the .envrc file.
	err = envrc.Generate(envrc.Config{
		ProviderSlug: chosenProvider.Slug(),
		FolderName:   chosenFolder.Name,
		FolderID:     chosenFolder.ID,
		Session:      session,
		Version:      version,
	})
	if err != nil {
		return fmt.Errorf("failed to write .envrc: %w", err)
	}

	PrintSuccess(".envrc created")

	// -- Step 8: Allow direnv so secrets load automatically --
	// Now that DIRENV_LOG_FORMAT="" is set globally (step 9 below) or was
	// already set from a previous init, we can safely allow the .envrc.
	// When the user's prompt returns, direnv's hook will silently load it.
	if err := envrc.AllowDirenv(); err != nil {
		// Non-fatal — direnv might not be installed.
		_ = err
	}

	// -- Step 9: Shell integration --
	// Install DIRENV_LOG_FORMAT="" (silence direnv) and the bwenv() shell
	// wrapper function into the user's shell RC file. The wrapper enables
	// commands like "bwenv allow", "bwenv disallow", "bwenv remove" to
	// modify the current shell's environment directly.
	userCfg, _ := config.Load()
	rcFile := ""
	rcModified := false

	// 9a: Silence direnv globally (unless user wants direnv output).
	if !userCfg.ShowDirenvOutput {
		silenceModified, silenceRC, silenceErr := envrc.SilenceDirenvGlobally()
		if silenceErr != nil {
			PrintInfo("Could not configure global direnv silence: " + silenceErr.Error())
		} else if silenceModified {
			PrintSuccess(fmt.Sprintf("Silenced direnv output in %s", silenceRC))
			rcFile = silenceRC
			rcModified = true
		}
	} else {
		PrintInfo("Direnv output is visible (configured via 'bwenv config')")
	}

	// 9b: Install the bwenv shell wrapper function.
	wrapperModified, wrapperRC, wrapperErr := envrc.InstallShellWrapper()
	if wrapperErr != nil {
		PrintInfo("Could not install shell wrapper: " + wrapperErr.Error())
	} else if wrapperModified {
		PrintSuccess(fmt.Sprintf("Installed bwenv shell wrapper in %s", wrapperRC))
		rcFile = wrapperRC
		rcModified = true
	}

	// -- Done! Show the final success summary --
	fmt.Println()
	printSuccessSummary(chosenProvider, chosenFolder, cwd, varNames, rcModified, rcFile)

	return nil
}

// printCancelled shows a clean cancellation message and exits.
func printCancelled() {
	fmt.Println()
	PrintWarning("Cancelled — no changes were made")
}

// printVariablePreview shows a compact, styled list of variable names that
// will be loaded from the chosen folder. Values are never shown.
func printVariablePreview(varNames []string) {
	count := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSuccess).
		Render(fmt.Sprintf("%s Found %d variable(s)", E("✓", "[OK]"), len(varNames)))

	fmt.Printf("  %s\n", count)

	// Show the variable names in a compact grid-like layout.
	// We indent each line and color the names with the secondary color.
	const maxPerLine = 4
	const maxShown = 16

	shown := varNames
	truncated := false
	if len(shown) > maxShown {
		shown = shown[:maxShown]
		truncated = true
	}

	for i := 0; i < len(shown); i += maxPerLine {
		end := i + maxPerLine
		if end > len(shown) {
			end = len(shown)
		}

		chunk := shown[i:end]
		styledNames := make([]string, len(chunk))
		for j, name := range chunk {
			styledNames[j] = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Render(name)
		}

		line := strings.Join(styledNames, lipgloss.NewStyle().
			Foreground(ColorMuted).Render("  "))
		fmt.Printf("    %s\n", line)
	}

	if truncated {
		remaining := len(varNames) - maxShown
		more := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Render(fmt.Sprintf("    ... and %d more", remaining))
		fmt.Println(more)
	}
}

// printNoProvidersHelp shows a helpful error message when no provider CLIs are installed.
// It lists all supported providers and how to install their CLI tools.
func printNoProvidersHelp(allProviders []provider.Provider) {
	fmt.Println()
	PrintBoxError(
		E("❌", "[ERROR]")+" No password manager CLI tools found!",
		"",
		"bwenv needs at least one of the following installed:",
	)
	fmt.Println()

	for _, p := range allProviders {
		fmt.Printf("  %s %s\n",
			CrossMark,
			lipgloss.NewStyle().Bold(true).Render(p.Name()),
		)
		fmt.Printf("      CLI command: %s\n",
			lipgloss.NewStyle().Foreground(ColorMuted).Render(p.CLICommand()),
		)
		fmt.Printf("      %s\n\n",
			lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).Render(p.Description()),
		)
	}
}

// printSuccessSummary displays the final success box after .envrc generation.
// Designed to be concise — one box with all info, clear next step.
func printSuccessSummary(p provider.Provider, folder *provider.Folder, cwd string, varNames []string, rcModified bool, rcFile string) {
	// Build a single summary box with everything the user needs.
	summaryLines := []string{
		E("✅", "[OK]") + " Setup complete!",
		"",
		fmt.Sprintf("  Provider:   %s", p.Name()),
		fmt.Sprintf("  Folder:     %s", folder.Name),
		fmt.Sprintf("  Variables:  %d secret(s)", len(varNames)),
		fmt.Sprintf("  Location:   %s/.envrc", ShortenHomePath(cwd)),
	}

	PrintBoxSuccess(summaryLines...)

	fmt.Println()

	if rcModified {
		// Shell RC was modified — user must source it (or restart) to
		// activate the bwenv wrapper and DIRENV_LOG_FORMAT.
		activateCmd := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).
			Render(fmt.Sprintf("source %s", rcFile))
		fmt.Fprintf(os.Stderr, "  %s  %s\n",
			lipgloss.NewStyle().Foreground(ColorMuted).Render("Activate now:"),
			activateCmd)

		subHint := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).
			Render("  After that, secrets load automatically when you cd into this directory.")
		fmt.Fprintln(os.Stderr, subHint)

		wrapperHint := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).
			Render("  Commands like bwenv allow/disallow/remove manage variables directly.")
		fmt.Fprintln(os.Stderr, wrapperHint)
	} else {
		// RC was already set up — everything works out of the box.
		hint := lipgloss.NewStyle().Foreground(ColorMuted).
			Render("Secrets load automatically when you cd into this directory.")
		fmt.Fprintf(os.Stderr, "  %s\n", hint)

		triggerHint := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).
			Render("  To load now: cd .")
		fmt.Fprintln(os.Stderr, triggerHint)
	}

	// If direnv is missing, show a warning.
	if _, err := exec.LookPath("direnv"); err != nil {
		fmt.Println()
		PrintBoxWarning(
			E("⚠ ", "! ")+" direnv is not installed",
			"",
			"   Install: https://direnv.net/",
		)
	}

	fmt.Println()
}

// formatProviderName returns the provider name styled with the secondary color.
func formatProviderName(name string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		Render(name)
}
