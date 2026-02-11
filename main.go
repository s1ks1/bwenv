// Package main is the entry point for the bwenv CLI application.
// bwenv helps you sync secrets from password managers (Bitwarden, 1Password)
// into your shell environment using direnv.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/s1ks1/bwenv/internal/check"
	"github.com/s1ks1/bwenv/internal/envrc"
	"github.com/s1ks1/bwenv/internal/ui"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	// Parse command from arguments, skipping any flags.
	args := os.Args[1:]
	command := ""
	flags := []string{}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		} else if command == "" {
			command = arg
		}
	}

	// Route to the appropriate command handler.
	switch command {
	case "init":
		// Full interactive TUI flow: pick provider → unlock vault → pick folder → generate .envrc
		runInit()

	case "export":
		// Non-interactive export for use inside .envrc files.
		// Usage: bwenv export --provider bitwarden --folder "MyFolder"
		runExport(args)

	case "remove":
		// Remove .envrc from the current directory.
		runRemove()

	case "test":
		// Check all dependencies and show a status report.
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
		ui.PrintSuccess(".envrc removed from current directory")
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

	fmt.Println("Usage: bwenv <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init          Interactive setup — pick provider, folder, and generate .envrc")
	fmt.Println("  export        Output env vars for use in .envrc (non-interactive)")
	fmt.Println("  remove        Remove .envrc from the current directory")
	fmt.Println("  test          Check dependencies and configuration")
	fmt.Println("  version       Show version")
	fmt.Println()
	fmt.Println("Export flags:")
	fmt.Println("  --provider    Secret provider: bitwarden, 1password")
	fmt.Println("  --folder      Folder/vault name to load secrets from")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  bwenv init")
	fmt.Println("  bwenv export --provider bitwarden --folder \"MySecrets\"")
	fmt.Println("  bwenv test")
	fmt.Println()
}
