// Package main is the entry point for the bwenv CLI application.
// bwenv helps you sync secrets from password managers (Bitwarden, 1Password)
// into your shell environment using direnv.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/check"
	"github.com/s1ks1/bwenv/internal/envrc"
	"github.com/s1ks1/bwenv/internal/ui"
)

// Version is set at build time via -ldflags.
// Overridden by GoReleaser or Makefile via: -ldflags "-X main.Version=v2.0.0"
var Version = "v2.0.0-dev"

func main() {
	// Parse command from arguments, skipping any flags.
	args := os.Args[1:]
	command := ""

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && command == "" {
			command = arg
		}
	}

	// Route to the appropriate command handler.
	switch command {
	case "init":
		// Full interactive TUI flow: pick provider → unlock vault → pick folder → generate .envrc
		runInit()

	case "export", "load":
		// Non-interactive export for use inside .envrc files.
		// "load" is an alias for "export" for convenience.
		// Usage: bwenv export --provider bitwarden --folder "MyFolder"
		runExport(args)

	case "remove", "clean":
		// Remove .envrc from the current directory.
		// "clean" is an alias for "remove" for convenience.
		runRemove()

	case "test", "doctor":
		// Check all dependencies and show a status report.
		// "doctor" is an alias for "test" (common CLI pattern).
		runTest()

	case "version", "--version", "-v":
		// Print the version string.
		fmt.Printf("bwenv %s\n", Version)

	default:
		// Show usage help when no command (or an unknown command) is given.
		printUsage()
	}
}

// runInit launches the full interactive TUI for setting up secrets.
func runInit() {
	if err := ui.RunInitFlow(Version); err != nil {
		ui.PrintError("Init failed", err)
		os.Exit(1)
	}
}

// runExport outputs "export KEY=VALUE" lines to stdout.
// This is designed to be called from within an .envrc file:
//
//	eval "$(bwenv export --provider bitwarden --folder MyFolder)"
func runExport(args []string) {
	provider, folder := parseExportFlags(args)

	if provider == "" || folder == "" {
		ui.PrintError("Missing flags", fmt.Errorf("both --provider and --folder are required"))
		fmt.Fprintln(os.Stderr, "Usage: bwenv export --provider <bitwarden|1password> --folder <name>")
		os.Exit(1)
	}

	if err := envrc.Export(provider, folder); err != nil {
		// Write errors to stderr so they don't pollute the eval output.
		fmt.Fprintf(os.Stderr, "bwenv export error: %v\n", err)
		os.Exit(1)
	}
}

// runRemove deletes the .envrc file in the current directory.
func runRemove() {
	removed, err := envrc.Remove()
	if err != nil {
		ui.PrintError("Remove failed", err)
		os.Exit(1)
	}

	if removed {
		ui.PrintSuccess("🗑  .envrc removed from current directory")
	} else {
		ui.PrintWarning("No .envrc found in current directory")
	}
}

// runTest checks all dependencies and prints a diagnostic report.
func runTest() {
	check.RunDiagnostics(Version)
}

// parseExportFlags extracts --provider and --folder values from the argument list.
func parseExportFlags(args []string) (provider, folder string) {
	for i := 0; i < len(args); i++ {
		switch {
		// Handle --provider=value or --provider value
		case args[i] == "--provider" && i+1 < len(args):
			i++
			provider = args[i]
		case strings.HasPrefix(args[i], "--provider="):
			provider = strings.TrimPrefix(args[i], "--provider=")

		// Handle --folder=value or --folder value
		case args[i] == "--folder" && i+1 < len(args):
			i++
			folder = args[i]
		case strings.HasPrefix(args[i], "--folder="):
			folder = strings.TrimPrefix(args[i], "--folder=")
		}
	}
	return provider, folder
}

// printUsage displays the help text with styled output.
func printUsage() {
	ui.PrintBanner(Version)

	// Styles for the help output.
	cmdStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"})
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})
	flagStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B21A8", Dark: "#C084FC"})
	exampleStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"})
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#E5E7EB"})

	fmt.Printf("  %s\n\n", headerStyle.Render("Usage: bwenv <command> [options]"))

	fmt.Printf("  %s\n\n", headerStyle.Render("Commands:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("init       "), descStyle.Render("🚀 Interactive setup — pick provider, folder, generate .envrc"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("export     "), descStyle.Render("📤 Output env vars for .envrc (non-interactive)"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("remove     "), descStyle.Render("🗑  Remove .envrc from the current directory"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("test       "), descStyle.Render("🩺 Check dependencies and configuration"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("version    "), descStyle.Render("📋 Show version"))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Export flags:"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--provider "), descStyle.Render("Secret provider: bitwarden, 1password"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--folder   "), descStyle.Render("Folder or vault name to load secrets from"))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Examples:"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv init"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv export --provider bitwarden --folder \"MySecrets\""))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv test"))
	fmt.Println()

	fmt.Printf("  %s\n\n", descStyle.Render("Aliases: load → export, clean → remove, doctor → test"))
}
