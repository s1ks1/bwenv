// Package main is the entry point for the bwenv CLI application.
// bwenv helps you sync secrets from password managers (Bitwarden, 1Password)
// into your shell environment using direnv.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
		// Note: init flow automatically runs allow at the end (best UX).
		runInit()

	case "export", "load":
		// Non-interactive export for use inside .envrc files.
		// "load" is an alias for "export" for convenience.
		// Usage: bwenv export --provider bitwarden --folder "MyFolder"
		runExport(args)

	case "allow":
		// Explicitly approve .envrc in the current directory.
		runAllow()

	case "disallow", "deny":
		// Explicitly block .envrc in the current directory.
		// "deny" is an alias for "disallow".
		runDisallow()

	case "examples":
		// Show copy-paste ready usage examples.
		runExamples()

	case "remove", "clean":
		// Remove .envrc from the current directory.
		// "clean" is an alias for "remove" for convenience.
		runRemove()

	case "config", "settings":
		// Interactive config editor for user preferences.
		// "settings" is an alias for "config" for convenience.
		runConfig()

	case "logout", "lock":
		// Lock all provider vaults and terminate sessions.
		// "lock" is an alias for "logout" for convenience.
		runLogout()

	case "status", "test", "doctor":
		// Comprehensive status and diagnostics view.
		// "test" and "doctor" are aliases for "status" (merged command).
		runStatus()

	case "version", "--version", "-v":
		// Print styled version information.
		runVersion()

	default:
		// Show usage help when no command (or an unknown command) is given.
		printUsage()
	}
}

// runVersion displays styled version information including version, license,
// author, and project URL — consistent with the rest of the bwenv UI.
func runVersion() {
	ui.PrintBanner(Version)

	mutedStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"})

	fmt.Printf("  %s  %s\n", labelStyle.Render("Version"), valueStyle.Render(Version))
	fmt.Printf("  %s  %s\n", labelStyle.Render("License"), valueStyle.Render("MIT"))
	fmt.Printf("  %s   %s\n", labelStyle.Render("Author"), valueStyle.Render("s1ks1"))
	fmt.Printf("  %s     %s\n", labelStyle.Render("Docs"), valueStyle.Render("https://github.com/s1ks1/bwenv"))
	fmt.Println()
	fmt.Printf("  %s\n\n", mutedStyle.Render("Run 'bwenv examples' for usage examples"))
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

// runAllow approves .envrc in the current directory via direnv and outputs
// export statements to stdout. When called through the bwenv shell wrapper
// (installed by "bwenv init"), the exports are eval'd automatically so
// variables appear in the current shell.
//
// If called directly in a TTY (without the wrapper), only approves .envrc
// and shows a hint — secrets are NOT printed to the terminal for safety.
func runAllow() {
	fi, _ := os.Stdout.Stat()
	isTTY := (fi.Mode() & os.ModeCharDevice) != 0

	if isTTY {
		// Direct invocation without the shell wrapper.
		// Don't print secrets to the terminal — just approve .envrc.
		if err := envrc.AllowDirenv(); err != nil {
			ui.PrintError("Allow failed", err)
			os.Exit(1)
		}
		prov, folder, _ := envrc.ParseEnvrcConfig()
		if prov != "" && folder != "" {
			fmt.Fprintf(os.Stderr, "  %s %s\n",
				ui.E("✅", "[OK]"),
				lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render(
					fmt.Sprintf(".envrc approved (%s / %s) — secrets load on next prompt", prov, folder)))
		} else {
			fmt.Fprintf(os.Stderr, "  %s %s\n",
				ui.E("✅", "[OK]"),
				lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render(
					".envrc approved — secrets load on next prompt"))
		}
		fmt.Fprintf(os.Stderr, "  %s\n",
			lipgloss.NewStyle().Foreground(ui.ColorMuted).Italic(true).Render(
				"Tip: restart your shell to enable the bwenv wrapper, then this works automatically."))
	} else {
		// Pipe mode (via shell wrapper or manual eval) — approve + export.
		_, _, err := envrc.AllowAndExport()
		if err != nil {
			ui.PrintError("Allow failed", err)
			os.Exit(1)
		}
	}
}

// runDisallow blocks .envrc in the current directory via direnv and
// outputs "unset VAR" statements to stdout. When called through the
// bwenv shell wrapper, the unsets are eval'd automatically so
// variables are cleared from the current shell.
func runDisallow() {
	varNames, err := envrc.DisallowAndUnset()
	if err != nil {
		ui.PrintError("Disallow failed", err)
		os.Exit(1)
	}
	if len(varNames) > 0 {
		fmt.Fprintf(os.Stderr, "  %s .envrc blocked — %d variable(s) cleared\n",
			ui.E("⛔", "[-]"), len(varNames))
	} else {
		fmt.Fprintf(os.Stderr, "  %s .envrc blocked\n", ui.E("⛔", "[-]"))
	}
}

// runExamples prints curated, copy-paste ready examples for common flows.
func runExamples() {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#E5E7EB"})
	exampleStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"})
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"}).Bold(true)

	ui.PrintBanner(Version)
	fmt.Println()

	// ── Quick Start ────────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(ui.E("🚀", ">>")+" Quick Start"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv init"), mutedStyle.Render("# Interactive setup — creates .envrc"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("cd ."), mutedStyle.Render("# Trigger direnv to load secrets"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("env | grep MY_VAR"), mutedStyle.Render("# Verify secrets are loaded"))
	fmt.Println()

	// ── Bitwarden Workflow ─────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(ui.E("🔑", ">>")+" Bitwarden Workflow"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 1: Create a folder in Bitwarden with custom fields"))
	fmt.Printf("    %s\n", mutedStyle.Render("         Each custom field → env var (field name = var name)"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 2: Set up bwenv in your project"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/your-project"))
	fmt.Printf("      %s\n", exampleStyle.Render("bwenv init"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 3: Non-interactive export (CI/scripts)"))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider bitwarden --folder \"MySecrets\")"`))
	fmt.Println()

	// ── 1Password Workflow ─────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(ui.E("🔐", ">>")+" 1Password Workflow"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 1: Create items in a 1Password vault"))
	fmt.Printf("    %s\n", mutedStyle.Render("         Item fields (label + value) → env vars"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 2: Set up bwenv in your project"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/your-project"))
	fmt.Printf("      %s\n", exampleStyle.Render("bwenv init"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 3: Non-interactive export (CI/scripts)"))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider 1password --folder \"Production\")"`))
	fmt.Println()

	// ── Direnv Control ─────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(ui.E("⚡", ">>")+ " Secret Management"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv allow"), mutedStyle.Render("# Approve .envrc + load secrets into shell"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv disallow"), mutedStyle.Render("# Block .envrc + clear variables from shell"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv remove"), mutedStyle.Render("# Delete .envrc + clear variables from shell"))
	fmt.Println()

	// ── Day-to-Day ─────────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(ui.E("📊", ">>")+" Day-to-Day"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv status"), mutedStyle.Render("# Full status + diagnostics"))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv config"), mutedStyle.Render("# Toggle emoji, direnv output, etc."))
	fmt.Printf("    %s  %s\n", exampleStyle.Render("bwenv logout"), mutedStyle.Render("# Lock vaults, end sessions"))
	fmt.Println()

	// ── Advanced ───────────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(ui.E("🔧", ">>")+" Advanced"))
	fmt.Printf("    %s\n", labelStyle.Render("Multiple projects with different vaults:"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/project-a && bwenv init   # Pick vault A"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/project-b && bwenv init   # Pick vault B"))
	fmt.Printf("    %s\n", mutedStyle.Render("    Secrets load automatically per directory!"))
	fmt.Println()
	fmt.Printf("    %s\n", labelStyle.Render("Use in shell scripts:"))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider bitwarden --folder \"Deploy\")"`))
	fmt.Printf("      %s\n", exampleStyle.Render("./deploy.sh   # $DB_URL, $API_KEY are now available"))
	fmt.Println()
	fmt.Printf("    %s\n", labelStyle.Render("CI/CD with 1Password service account:"))
	fmt.Printf("      %s\n", exampleStyle.Render("export OP_SERVICE_ACCOUNT_TOKEN=\"...\""))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider 1password --folder \"CI\")"`))
	fmt.Println()

	fmt.Printf("  %s\n\n", mutedStyle.Render("For installation instructions, see: INSTALL.md"))
}

// runRemove deletes .envrc and .bwenv_vars, outputs "unset VAR" statements
// to stdout. When called through the bwenv shell wrapper, the unsets are
// eval'd automatically so variables are cleared from the current shell.
func runRemove() {
	removed, varNames, err := envrc.RemoveAndUnset()
	if err != nil {
		ui.PrintError("Remove failed", err)
		os.Exit(1)
	}

	if removed {
		if len(varNames) > 0 {
			fmt.Fprintf(os.Stderr, "  %s .envrc removed — %d variable(s) cleared\n",
				ui.E("🗑 ", "[-]"), len(varNames))
		} else {
			fmt.Fprintf(os.Stderr, "  %s .envrc removed\n", ui.E("🗑 ", "[-]"))
		}
	} else {
		fmt.Fprintf(os.Stderr, "  %s No .envrc found in current directory\n",
			ui.E("⚠ ", "!"))
	}
}

// runConfig launches the interactive configuration editor.
func runConfig() {
	if err := ui.RunConfigFlow(Version); err != nil {
		ui.PrintError("Config failed", err)
		os.Exit(1)
	}
}

// runLogout locks all provider vaults and terminates active sessions.
func runLogout() {
	if err := ui.RunLogoutFlow(Version); err != nil {
		ui.PrintError("Logout failed", err)
		os.Exit(1)
	}
}

// runStatus displays a comprehensive overview of the current bwenv state,
// including diagnostics. This is the merged status + test command.
func runStatus() {
	if err := ui.RunStatusFlow(Version); err != nil {
		ui.PrintError("Status failed", err)
		os.Exit(1)
	}
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

	fmt.Printf("  %s\n\n", headerStyle.Render("Setup:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("init       "), descStyle.Render(ui.E("🚀", "->")+` Interactive setup — pick provider, folder, generate .envrc`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("export     "), descStyle.Render(ui.E("📤", "->")+` Output env vars for .envrc (non-interactive)`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Secret Management:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("allow      "), descStyle.Render(ui.E("✅", "->")+` Approve .envrc and load secrets into shell`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("disallow   "), descStyle.Render(ui.E("⛔", "->")+` Block .envrc and clear variables from shell`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("remove     "), descStyle.Render(ui.E("🗑 ", "->")+`  Delete .envrc and clear variables from shell`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Diagnostics & Config:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("status     "), descStyle.Render(ui.E("📊", "->")+` Full status overview and diagnostics`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("config     "), descStyle.Render(ui.E("⚙ ", "->")+`  Configure preferences (emoji, direnv output, etc.)`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("logout     "), descStyle.Render(ui.E("🔒", "->")+` Lock vaults and terminate active sessions`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Help:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("examples   "), descStyle.Render(ui.E("📚", "->")+` Show detailed usage examples`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("version    "), descStyle.Render(ui.E("📋", "->")+` Show version`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Export flags:"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--provider "), descStyle.Render("Secret provider: bitwarden, 1password"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--folder   "), descStyle.Render("Folder or vault name to load secrets from"))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Quick Start:"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv init            # Interactive setup"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv status          # Check everything is working"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv examples        # See all usage examples"))
	fmt.Println()

	fmt.Printf("  %s\n\n", descStyle.Render("Aliases: load → export, clean → remove, doctor/test → status, lock → logout, deny → disallow, settings → config"))
}
