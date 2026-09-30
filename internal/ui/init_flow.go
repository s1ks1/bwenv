// Package ui — init flow orchestrator.
// This file ties together all the TUI components (provider picker, folder picker,
// authentication, and .envrc generation) into a single interactive flow.
// It is the main entry point for the "bwenv init" command.
//
// The flow now also:
//   - Previews which variables will be loaded (showing names, not values)
//   - Automatically runs "direnv allow" so the user doesn't see the scary
//     "direnv: error .envrc is blocked" message
//   - Shows a beautiful final summary with emoji (configurable)
//   - Respects user config for direnv output silencing
package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/activation/direnv"
	activationshell "github.com/s1ks1/bwenv/v3/internal/activation/shell"
	"github.com/s1ks1/bwenv/v3/internal/config"
	"github.com/s1ks1/bwenv/v3/internal/export"
	"github.com/s1ks1/bwenv/v3/internal/project"
	"github.com/s1ks1/bwenv/v3/internal/provider"
	"github.com/s1ks1/bwenv/v3/internal/shell"
)

// RunInitFlow executes the full interactive initialization process for the
// given activation backend. The saved default is used for new projects;
// existing project metadata and explicit --activation take precedence.
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
func RunInitFlow(version string, activationMode string) error {
	ctx := context.Background()
	const totalSteps = 7

	PrintBanner(version)
	userCfg, err := config.Load()
	if err != nil {
		return err
	}
	if activationMode == "" {
		activationMode = userCfg.ActivationMode
		if cfg, err := project.Load("."); err == nil {
			activationMode = cfg.Activation.Mode
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	activator, err := activation.Get(activationMode)
	if err != nil {
		return err
	}
	activationMode = activator.Name()
	if !activator.Available() {
		return fmt.Errorf("%s is not installed; install it or select --activation shell", activationMode)
	}
	PrintInfo("Activation: " + activationMode + " — change the default with 'bwenv config'; override this project with --activation")
	if activationMode == "shell" {
		PrintInfo("Native shell hook (experimental): no direnv or mise required.")
	}

	// -- Step 1: Gather all registered providers --
	allProviders := provider.All()
	if len(allProviders) == 0 {
		return fmt.Errorf("no secret providers are registered — this is a bug in bwenv")
	}

	available := provider.Available()
	if len(available) == 0 {
		printNoProvidersHelp(allProviders)
		return fmt.Errorf("no supported password manager CLI tools found on this system")
	}

	// -- Step 2: Provider selection --
	var chosenProvider provider.Provider

	if len(available) == 1 {
		chosenProvider = available[0]
		PrintStep(1, totalSteps, fmt.Sprintf("%s Using %s (only available provider)", E("🔑", "[>]"), formatProviderName(chosenProvider.Name())))
		fmt.Println()
	} else {
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
			return ErrCancelled
		}

		chosenProvider = result.Chosen()
		if chosenProvider == nil {
			return fmt.Errorf("no provider was selected")
		}

		fmt.Println()
		PrintSuccess(fmt.Sprintf("Selected: %s", chosenProvider.Name()))
	}

	authProvider, err := provider.AsAuthenticator(chosenProvider)
	if err != nil {
		return fmt.Errorf("provider %s is not usable: %w", chosenProvider.Name(), err)
	}
	folderLister, err := provider.AsFolderLister(chosenProvider)
	if err != nil {
		return fmt.Errorf("provider %s is not usable: %w", chosenProvider.Name(), err)
	}
	fetcher, err := provider.AsSecretFetcher(chosenProvider)
	if err != nil {
		return fmt.Errorf("provider %s is not usable: %w", chosenProvider.Name(), err)
	}

	// -- Step 3: Authenticate with the provider --
	PrintStep(2, totalSteps, E("🔓", "[>]")+" Authenticating with "+chosenProvider.Name()+"...")
	fmt.Println()

	session, err := authProvider.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	PrintSuccess(E("🔓", "->") + " Vault unlocked")
	fmt.Println()

	// -- Step 4: Fetch the folder list --
	PrintStep(3, totalSteps, E("📂", "[>]")+" Fetching folders from "+chosenProvider.Name()+"...")

	folders, err := folderLister.ListFolders(ctx, session)
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
		return ErrCancelled
	}

	chosenFolder := folderResult.Chosen()
	if chosenFolder == nil {
		return fmt.Errorf("no folder was selected")
	}

	fmt.Println()
	PrintSuccess(fmt.Sprintf("Selected: %s", chosenFolder.Name))
	fmt.Println()

	// -- Step 6: Pick specific items within the folder --
	PrintStep(5, totalSteps, E("🎯", "[>]")+" Pick individual items (space to toggle, a = all)")
	fmt.Println()

	var itemIDs []string
	var itemNames []string

	items, listErr := folderLister.ListItems(ctx, session, *chosenFolder)
	if listErr != nil {
		PrintWarning(fmt.Sprintf("Could not list items: %v", listErr))
		PrintInfo("All items in the folder will be loaded instead.")
		fmt.Println()
	} else if len(items) == 0 {
		PrintInfo("No items found in this folder.")
		fmt.Println()
	} else if len(items) == 1 {
		itemIDs = append(itemIDs, items[0].ID)
		itemNames = append(itemNames, items[0].Name)
		PrintSuccess(fmt.Sprintf("Using 1 item: %s", items[0].Name))
		fmt.Println()
	} else {
		secretPicker := NewSecretPicker(items, chosenProvider.Name(), chosenFolder.Name)
		secretProgram := tea.NewProgram(secretPicker)

		finalSecretModel, err := secretProgram.Run()
		if err != nil {
			PrintWarning(fmt.Sprintf("Item picker failed: %v", err))
			PrintInfo("All items in the folder will be loaded instead.")
			fmt.Println()
		} else {
			secretResult := finalSecretModel.(SecretPickerModel)
			if secretResult.Cancelled() {
				printCancelled()
				return ErrCancelled
			}

			chosenItems := secretResult.Selected()
			if len(chosenItems) == 0 {
				PrintWarning("No items selected — all items in the folder will be loaded.")
				fmt.Println()
			} else {
				for _, item := range chosenItems {
					itemIDs = append(itemIDs, item.ID)
					itemNames = append(itemNames, item.Name)
				}
				PrintSuccess(fmt.Sprintf("Selected %d item(s)", len(chosenItems)))
				fmt.Println()
			}
		}
	}

	// -- Step 7: Preview secrets (show variable names, not values) --
	PrintStep(6, totalSteps, E("🔍", "[>]")+" Scanning secrets...")

	var varNames []string
	if len(itemIDs) > 0 {
		varNames, err = export.PreviewSecretsByIDs(ctx, fetcher, session, *chosenFolder, itemIDs)
	} else {
		varNames, err = export.PreviewSecrets(ctx, fetcher, session, *chosenFolder)
	}
	if err != nil {
		PrintWarning(fmt.Sprintf("Could not preview secrets: %v", err))
		PrintInfo("Project configuration will still be generated; run bwenv login to load secrets.")
		fmt.Println()
	} else if len(varNames) == 0 {
		PrintWarning("No secrets found")
		PrintInfo("Make sure your vault items have custom fields with names and values.")
		fmt.Println()
	} else {
		printVariablePreview(varNames)
		fmt.Println()
	}

	// -- Step 8: Generate the .envrc file --
	PrintStep(7, totalSteps, E("📝", "[>]")+" Configuring "+activationMode+" activation...")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	err = activator.Install(activation.Config{
		ProviderSlug: chosenProvider.Slug(),
		FolderName:   chosenFolder.Name,
		FolderID:     chosenFolder.ID,
		Version:      version,
		ItemIDs:      itemIDs,
		ItemNames:    itemNames,
	})
	if err != nil {
		return fmt.Errorf("failed to configure %s activation: %w", activationMode, err)
	}

	PrintSuccess(activationMode + " activation configured")

	// -- Step 8: Approve the activation artifact so secrets load automatically --
	// When the user's prompt returns, the backend hook will load it silently.
	if err := activator.Approve(); err != nil {
		PrintWarning("Activation approval failed: " + err.Error())
	}

	// -- Step 9: Shell integration --
	// Install DIRENV_LOG_FORMAT="" (silence direnv) and the bwenv() shell
	// wrapper function into the user's shell RC file. The wrapper enables
	// commands like "bwenv allow", "bwenv disallow", "bwenv remove" to
	// modify the current shell's environment directly.

	rcFile := ""
	rcModified := false
	if activationMode == "shell" {
		if rc, err := shell.DetectRC(); err == nil {
			rcFile = shell.ShortenHomePath(rc)
			rcModified = true
		}
	}

	// 9a: Silence direnv globally (unless user wants direnv output). This is
	// direnv-specific; other backends have no equivalent noise.
	if activationMode == "direnv" && !userCfg.ShowDirenvOutput {
		silenceModified, silenceRC, silenceErr := direnv.SilenceGlobally()
		if silenceErr != nil {
			PrintInfo("Could not configure global direnv silence: " + silenceErr.Error())
		} else if silenceModified {
			PrintSuccess(fmt.Sprintf("Silenced direnv output in %s", silenceRC))
			rcFile = silenceRC
			rcModified = true
		}
	} else if activationMode == "direnv" {
		PrintInfo("Direnv output is visible (configured via 'bwenv config')")
	}

	// 9b: Install the bwenv shell wrapper function.
	wrapperModified, wrapperRC, wrapperErr := shell.InstallWrapper()
	if wrapperErr != nil {
		PrintInfo("Could not install shell wrapper: " + wrapperErr.Error())
	} else if wrapperModified {
		PrintSuccess(fmt.Sprintf("Installed bwenv shell wrapper in %s", wrapperRC))
		rcFile = wrapperRC
		rcModified = true
	}

	// -- Done! Show the final success summary --
	fmt.Println()
	printSuccessSummary(chosenProvider, chosenFolder, itemNames, cwd, varNames, rcModified, rcFile, activationMode)

	return nil
}

var ErrCancelled = errors.New("cancelled")

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
		Render(fmt.Sprintf("%s Found %d variable(s)", E("✅", "[OK]"), len(varNames)))

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
func printSuccessSummary(p provider.Provider, folder *provider.Folder, itemNames []string, cwd string, varNames []string, rcModified bool, rcFile string, activationMode string) {
	summaryLines := []string{
		E("✅", "[OK]") + " Setup complete!",
		"",
		fmt.Sprintf("  Provider:   %s", p.Name()),
		fmt.Sprintf("  Folder:     %s", folder.Name),
	}

	if len(itemNames) > 0 {
		if len(itemNames) <= 3 {
			summaryLines = append(summaryLines, fmt.Sprintf("  Items:      %s", strings.Join(itemNames, ", ")))
		} else {
			summaryLines = append(summaryLines, fmt.Sprintf("  Items:      %d selected", len(itemNames)))
		}
	} else {
		summaryLines = append(summaryLines, "  Items:      All")
	}

	summaryLines = append(summaryLines,
		fmt.Sprintf("  Variables:  %d secret(s)", len(varNames)),
		fmt.Sprintf("  Location:   %s/.bwenv.toml", ShortenHomePath(cwd)),
		fmt.Sprintf("  Activation: %s", activationMode),
	)

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
			Render("  Then run 'bwenv login' to load secrets into this shell.")
		fmt.Fprintln(os.Stderr, subHint)

		wrapperHint := lipgloss.NewStyle().Foreground(ColorMuted).Italic(true).
			Render("  Commands like bwenv allow/disallow/remove manage variables directly.")
		fmt.Fprintln(os.Stderr, wrapperHint)
	} else {
		// The session produced during init belongs to this process, not the shell.
		hint := lipgloss.NewStyle().Foreground(ColorMuted).
			Render("Run 'bwenv login' to authenticate and load secrets into this shell.")
		fmt.Fprintf(os.Stderr, "  %s\n", hint)
	}

	printActivationInstructions(activationMode)

	fmt.Println()
}

// formatProviderName returns the provider name styled with the secondary color.
func formatProviderName(name string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		Render(name)
}

func printActivationInstructions(mode string) {
	name := activationshell.DetectShell(os.Getenv("SHELL"))
	switch mode {
	case "shell":
		PrintInfo("Recommended for Bash/Zsh/Fish: native hook, no additional CLI required.")
		if name == "fish" {
			PrintInfo("Manual hook setup: bwenv hook fish | source")
		} else {
			PrintInfo("Manual hook setup: eval \"$(bwenv hook " + name + ")\"")
		}
		PrintInfo("After bwenv init, source the displayed shell RC file or open a new terminal, then run 'bwenv login'.")
	case "direnv", "mise":
		PrintInfo("Install " + mode + " and enable its shell hook once in your shell RC file:")
		if name == "fish" {
			command := "direnv hook fish | source"
			if mode == "mise" {
				command = "mise activate fish | source"
			}
			PrintInfo(command)
		} else {
			command := "direnv hook " + name
			if mode == "mise" {
				command = "mise activate " + name
			}
			PrintInfo("eval \"$(" + command + ")\"")
		}
		PrintInfo("Open a new terminal or source your RC file; run 'bwenv login' in the project.")
		if mode == "mise" {
			PrintInfo("Trust the project once: mise trust")
		}
	}
}
