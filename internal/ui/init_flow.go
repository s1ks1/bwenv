// Package ui — init flow orchestrator.
// This file ties together all the TUI components (provider picker, folder picker,
// authentication, and .envrc generation) into a single interactive flow.
// It is the main entry point for the "bwenv init" command.
//
// The flow now also:
//   - Previews which variables will be loaded (showing names, not values)
//   - Automatically runs "direnv allow" so the user doesn't see the scary
//     "direnv: error .envrc is blocked" message
//   - Shows a beautiful final summary with emojis
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		PrintStep(1, totalSteps, fmt.Sprintf("🔑 Using %s (only available provider)", formatProviderName(chosenProvider.Name())))
		fmt.Println()
	} else {
		// Launch the interactive provider picker TUI.
		PrintStep(1, totalSteps, "🔑 Select a secret provider")
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
	PrintStep(2, totalSteps, "🔓 Authenticating with "+chosenProvider.Name()+"...")
	fmt.Println()

	session, err := chosenProvider.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	PrintSuccess("Vault unlocked")
	fmt.Println()

	// -- Step 4: Fetch the folder list --
	PrintStep(3, totalSteps, "📂 Fetching folders from "+chosenProvider.Name()+"...")

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
	PrintStep(4, totalSteps, "📁 Pick a folder to load secrets from")
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
	PrintStep(5, totalSteps, "🔍 Scanning secrets in folder...")

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
	PrintStep(6, totalSteps, "📝 Generating .envrc...")

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

	// -- Step 8: Auto-run "direnv allow" --
	// This prevents the scary "direnv: error .envrc is blocked" message.
	direnvAllowed := false
	if err := envrc.AllowDirenv(); err != nil {
		// Non-fatal — just tell the user to do it manually.
		PrintInfo("Run 'direnv allow' to approve the .envrc file")
	} else {
		direnvAllowed = true
		PrintSuccess("direnv allow — approved automatically")
	}

	// -- Step 9: Silence direnv globally --
	// Add DIRENV_LOG_FORMAT="" to the user's shell RC so that ALL direnv
	// messages are suppressed — including "direnv: loading .envrc" which
	// is printed BEFORE the .envrc runs and can't be silenced from within it.
	modified, rcFile, silenceErr := envrc.SilenceDirenvGlobally()
	if silenceErr != nil {
		// Non-fatal — the .envrc still works, just with some direnv noise on first load.
		PrintInfo("Could not configure global direnv silence: " + silenceErr.Error())
	} else if modified {
		PrintSuccess(fmt.Sprintf("Silenced direnv output in %s", rcFile))
	}
	// If !modified && silenceErr == nil, the line was already present — nothing to report.

	// -- Done! Show the final success summary --
	fmt.Println()
	printSuccessSummary(chosenProvider, chosenFolder, cwd, varNames, direnvAllowed, modified, rcFile)

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
		Render(fmt.Sprintf("✓ Found %d variable(s)", len(varNames)))

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
		"❌ No password manager CLI tools found!",
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
// It shows what was configured, which variables will load, and what happens next.
func printSuccessSummary(p provider.Provider, folder *provider.Folder, cwd string, varNames []string, direnvAllowed bool, rcModified bool, rcFile string) {
	// Build the summary lines for the success box.
	summaryLines := []string{
		"✅ Setup complete!",
		"",
		fmt.Sprintf("  Provider:   %s", p.Name()),
		fmt.Sprintf("  Folder:     %s", folder.Name),
		fmt.Sprintf("  Location:   %s/.envrc", cwd),
	}

	if len(varNames) > 0 {
		summaryLines = append(summaryLines,
			fmt.Sprintf("  Variables:  %d secret(s) ready to load", len(varNames)))
	}

	if rcModified {
		summaryLines = append(summaryLines,
			fmt.Sprintf("  Shell RC:   %s (direnv silenced)", rcFile))
	}

	PrintBoxSuccess(summaryLines...)

	// If direnv was allowed, tell the user things are ready.
	// If not, tell them what to do.
	if direnvAllowed {
		fmt.Println()
		readyTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess).
			Render("⚡ Your secrets are ready!")

		fmt.Printf("  %s\n\n", readyTitle)

		hint1 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Open a new terminal or run")
		cmd1 := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("cd .")
		hint1b := lipgloss.NewStyle().Foreground(ColorMuted).Render("to trigger direnv")
		fmt.Printf("    %s %s %s\n", hint1, cmd1, hint1b)

		hint2 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Verify with")
		cmd2 := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("env | grep <YOUR_VAR>")
		fmt.Printf("    %s %s\n", hint2, cmd2)

		hint3 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Remove when done:")
		cmd3 := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("bwenv remove")
		fmt.Printf("    %s %s\n", hint3, cmd3)
	} else {
		fmt.Println()
		nextTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Render("Next steps:")

		fmt.Printf("  %s\n\n", nextTitle)

		step1 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Approve the .envrc file:")
		cmd1 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("direnv allow")
		fmt.Printf("    1. %s  %s\n", step1, cmd1)

		step2 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Verify secrets are loaded:")
		cmd2 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("env | grep <YOUR_VAR>")
		fmt.Printf("    2. %s  %s\n", step2, cmd2)

		step3 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Remove secrets when done:")
		cmd3 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("bwenv remove")
		fmt.Printf("    3. %s  %s\n", step3, cmd3)
	}

	fmt.Println()

	// Quiet hint about what happens behind the scenes.
	hint := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render("  Secrets load automatically when you cd into this directory.")
	fmt.Println(hint)

	if rcModified {
		rcHint := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Render(fmt.Sprintf("  Restart your shell or run: source %s", rcFile))
		fmt.Println(rcHint)
	}

	// If direnv is not installed, show a helpful nudge.
	if _, err := exec.LookPath("direnv"); err != nil {
		fmt.Println()
		PrintBoxWarning(
			"⚠  direnv is not installed",
			"",
			"   bwenv generates .envrc files that direnv loads automatically.",
			"   Install direnv: https://direnv.net/",
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
