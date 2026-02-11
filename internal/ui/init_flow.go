// Package ui — init flow orchestrator.
// This file ties together all the TUI components (provider picker, folder picker,
// authentication, and .envrc generation) into a single interactive flow.
// It is the main entry point for the "bwenv init" command.
package ui

import (
	"fmt"
	"os"
	"path/filepath"

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
//  6. Generate a .envrc file in the current directory.
//
// Returns an error if any step fails or if the user cancels.
func RunInitFlow(version string) error {
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
		PrintInfo(fmt.Sprintf("Using %s (only available provider)", chosenProvider.Name()))
		fmt.Println()
	} else {
		// Launch the interactive provider picker TUI.
		pickerModel := NewProviderPicker(allProviders)
		program := tea.NewProgram(pickerModel)

		finalModel, err := program.Run()
		if err != nil {
			return fmt.Errorf("provider picker failed: %w", err)
		}

		result := finalModel.(ProviderPickerModel)
		if result.Cancelled() {
			fmt.Println()
			PrintWarning("Cancelled by user")
			os.Exit(0)
		}

		chosenProvider = result.Chosen()
		if chosenProvider == nil {
			return fmt.Errorf("no provider was selected")
		}
	}

	fmt.Println()
	PrintStep(1, 4, fmt.Sprintf("Selected provider: %s", formatProviderName(chosenProvider.Name())))

	// -- Step 3: Authenticate with the provider --
	PrintStep(2, 4, "Authenticating...")
	fmt.Println()

	session, err := chosenProvider.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	PrintSuccess("Authentication successful")
	fmt.Println()

	// -- Step 4: Fetch the folder list --
	PrintStep(3, 4, "Fetching folders...")

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
	folderModel := NewFolderPicker(folders, chosenProvider.Name())
	folderProgram := tea.NewProgram(folderModel)

	finalFolderModel, err := folderProgram.Run()
	if err != nil {
		return fmt.Errorf("folder picker failed: %w", err)
	}

	folderResult := finalFolderModel.(FolderPickerModel)
	if folderResult.Cancelled() {
		fmt.Println()
		PrintWarning("Cancelled by user")
		os.Exit(0)
	}

	chosenFolder := folderResult.Chosen()
	if chosenFolder == nil {
		return fmt.Errorf("no folder was selected")
	}

	fmt.Println()
	PrintSuccess(fmt.Sprintf("Selected folder: %s", chosenFolder.Name))

	// -- Step 6: Generate the .envrc file --
	PrintStep(4, 4, "Generating .envrc...")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	envrcPath := filepath.Join(cwd, ".envrc")

	// Check if .envrc already exists and warn the user.
	if _, statErr := os.Stat(envrcPath); statErr == nil {
		fmt.Println()
		PrintWarning(".envrc already exists — it will be overwritten")
	}

	// Build the .envrc content. The generated file calls "bwenv export" which
	// handles authentication and secret retrieval at load time.
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

	// -- Done! Show the success summary --
	fmt.Println()
	printSuccessSummary(chosenProvider, chosenFolder, cwd)

	return nil
}

// printNoProvidersHelp shows a helpful error message when no provider CLIs are installed.
// It lists all supported providers and how to install their CLI tools.
func printNoProvidersHelp(allProviders []provider.Provider) {
	fmt.Println()
	PrintBoxError(
		"No password manager CLI tools found!",
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
// It shows what was configured, where the file was written, and what to do next.
func printSuccessSummary(p provider.Provider, folder *provider.Folder, cwd string) {
	providerTag := FormatProviderTag(p.Slug())

	// Build a nice summary box.
	PrintBoxSuccess(
		"✓ .envrc generated successfully!",
		"",
		fmt.Sprintf("  Provider:  %s", p.Name()),
		fmt.Sprintf("  Folder:    %s", folder.Name),
		fmt.Sprintf("  Location:  %s/.envrc", cwd),
	)

	fmt.Println()

	// Next steps hint.
	nextStepsTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("Next steps:")

	fmt.Printf("  %s\n\n", nextStepsTitle)

	step1 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Allow direnv to load the file:")
	cmd1 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("direnv allow")
	fmt.Printf("    1. %s  %s\n", step1, cmd1)

	step2 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Verify secrets are loaded:")
	cmd2 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("env | grep <YOUR_VAR>")
	fmt.Printf("    2. %s  %s\n", step2, cmd2)

	step3 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Remove secrets when done:")
	cmd3 := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("bwenv remove")
	fmt.Printf("    3. %s  %s\n", step3, cmd3)

	fmt.Println()

	// Reminder about the provider tag in .envrc.
	hint := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render(fmt.Sprintf("  Your .envrc uses provider %s — secrets load automatically via direnv.", providerTag))
	fmt.Println(hint)
	fmt.Println()
}

// formatProviderName returns the provider name styled with the secondary color.
func formatProviderName(name string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		Render(name)
}
