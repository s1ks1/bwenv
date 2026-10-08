package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func PrintVersion(version string) {
	PrintBanner(version)

	mutedStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"})

	fmt.Printf("  %s  %s\n", labelStyle.Render("Version"), valueStyle.Render(version))
	fmt.Printf("  %s  %s\n", labelStyle.Render("License"), valueStyle.Render("MIT"))
	fmt.Printf("  %s   %s\n", labelStyle.Render("Author"), valueStyle.Render("s1ks1"))
	fmt.Printf("  %s     %s\n", labelStyle.Render("Docs"), valueStyle.Render("https://github.com/s1ks1/bwenv"))
	fmt.Println()
	fmt.Printf("  %s\n\n", mutedStyle.Render("Run 'bwenv examples' for usage examples"))
}

func PrintExamples(version string) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#E5E7EB"})
	exampleStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"})
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"}).Bold(true)
	exampleColumn := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}).
		Width(18)
	printExample := func(command string, comment string) {
		fmt.Printf("    %s %s\n", exampleColumn.Render(command), mutedStyle.Render("# "+comment))
	}

	PrintBanner(version)
	fmt.Println()

	// ── Quick Start ────────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(E("🚀", ">>")+" Quick Start"))
	printExample("bwenv init", "Interactive setup — choose provider and project")
	printExample("bwenv doctor", "Check setup and diagnose common issues")
	printExample("cd .", "Trigger the configured hook")
	printExample("bwenv status", "Verify setup without showing values")
	fmt.Println()

	// ── Bitwarden Workflow ─────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(E("🔑", ">>")+" Bitwarden Workflow"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 1: Create a folder in Bitwarden with custom fields"))
	fmt.Printf("    %s\n", mutedStyle.Render("         Each custom field → env var (field name = var name)"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 2: Set up bwenv in your project"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/your-project"))
	fmt.Printf("      %s\n", exampleStyle.Render("bwenv init"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 3: Non-interactive export (CI/scripts)"))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider bitwarden --folder "MySecrets")"`))
	fmt.Println()

	// ── 1Password Workflow ─────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(E("🔐", ">>")+" 1Password Workflow"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 1: Create items in a 1Password vault"))
	fmt.Printf("    %s\n", mutedStyle.Render("         Item fields (label + value) → env vars"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 2: Set up bwenv in your project"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/your-project"))
	fmt.Printf("      %s\n", exampleStyle.Render("bwenv init"))
	fmt.Printf("    %s\n", labelStyle.Render("Step 3: Non-interactive export (CI/scripts)"))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider 1password --folder "Production")"`))
	fmt.Println()

	// ── Direnv Control ─────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(E("⚡", ">>")+" Secret Management"))
	printExample("bwenv allow", "Enable project + load secrets into shell")
	printExample("bwenv disallow", "Disable project + restore shell variables")
	printExample("bwenv remove", "Remove project config + restore variables")
	fmt.Println()

	// ── Day-to-Day ─────────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(E("📊", ">>")+" Day-to-Day"))
	printExample("bwenv login", "Re-authenticate when session expires")
	printExample("bwenv refresh", "Sync and reload project secrets")
	printExample("bwenv doctor", "Safe preflight diagnostics")
	printExample("bwenv status", "Full status + diagnostics")
	printExample("bwenv config", "Toggle emoji, direnv output, etc.")
	printExample("bwenv logout", "Lock vaults, end sessions")
	fmt.Println()

	// ── Advanced ───────────────────────────────────────────
	fmt.Printf("  %s\n\n", headerStyle.Render(E("🔧", ">>")+" Advanced"))
	fmt.Printf("    %s\n", labelStyle.Render("Multiple projects with different vaults:"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/project-a && bwenv init   # Pick vault A"))
	fmt.Printf("      %s\n", exampleStyle.Render("cd ~/project-b && bwenv init   # Pick vault B"))
	fmt.Printf("    %s\n", mutedStyle.Render("    Secrets load automatically per directory!"))
	fmt.Println()
	fmt.Printf("    %s\n", labelStyle.Render("Use in shell scripts:"))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider bitwarden --folder "Deploy")"`))
	fmt.Printf("      %s\n", exampleStyle.Render("./deploy.sh   # $DB_URL, $API_KEY are now available"))
	fmt.Println()
	fmt.Printf("    %s\n", labelStyle.Render("CI/CD with 1Password service account:"))
	fmt.Printf("      %s\n", exampleStyle.Render("export OP_SERVICE_ACCOUNT_TOKEN=\"...\""))
	fmt.Printf("      %s\n", exampleStyle.Render(`eval "$(bwenv export --provider 1password --folder "CI")"`))
	fmt.Println()

	fmt.Printf("  %s\n\n", mutedStyle.Render("For installation instructions, see: INSTALL.md"))
}

func PrintUsage(version string) {
	PrintBanner(version)

	// Styles for the help output.
	cmdStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"})
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})
	flagStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B21A8", Dark: "#C084FC"})
	exampleStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"})
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#E5E7EB"})

	fmt.Printf("  %s\n\n", headerStyle.Render("Usage: bwenv <command> [options]"))

	fmt.Printf("  %s\n\n", headerStyle.Render("Setup:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("init       "), descStyle.Render(E("🚀", "->")+` Interactive setup — pick provider, folder and hook`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("export     "), descStyle.Render(E("📤", "->")+` Output shell assignments (non-interactive)`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("activate   "), descStyle.Render(E("⚡", "->")+` Make the project's activation artifact ready`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("deactivate "), descStyle.Render(E("⛔", "->")+` Revoke activation and clear variables`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("hook       "), descStyle.Render(E("🪝", "->")+` Print a native shell hook (experimental; zsh, bash, fish)`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Secret Management:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("allow      "), descStyle.Render(E("✅", "->")+` Enable project and load secrets into shell`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("disallow   "), descStyle.Render(E("⛔\ufe0f", "->")+` Disable project and restore shell variables`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("remove     "), descStyle.Render(E("🗑️", "->")+` Remove project config and restore variables`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Diagnostics & Config:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("login      "), descStyle.Render(E("🔓", "->")+` Re-authenticate and reload secrets (session expired?)`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("refresh    "), descStyle.Render(E("🔄", "->")+` Refresh provider data and reload the environment`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("migrate    "), descStyle.Render(E("🔁", "->")+` Convert a legacy project (use --dry-run to preview)`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("status     "), descStyle.Render(E("📊", "->")+` Full status overview and diagnostics`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("doctor     "), descStyle.Render(E("🩺", "->")+` Run safe, shareable setup checks`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("benchmark  "), descStyle.Render(E("⏱️", "->")+` Measure provider calls for this project`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("config     "), descStyle.Render(E("⚙️ ", "->")+`  Configure preferences (emoji, direnv output, etc.)`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("logout     "), descStyle.Render(E("🔒", "->")+` Lock vaults and terminate active sessions`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Help:"))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("examples   "), descStyle.Render(E("📚", "->")+` Show detailed usage examples`))
	fmt.Printf("    %s   %s\n", cmdStyle.Render("version    "), descStyle.Render(E("📋", "->")+` Show version`))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Init flags:"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--activation"), descStyle.Render("Backend override: shell (default, experimental), direnv, mise (experimental); set default in config"))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Export flags:"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--provider "), descStyle.Render("Secret provider: bitwarden, 1password"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--folder   "), descStyle.Render("Folder or vault name to load secrets from"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--folder-id"), descStyle.Render("Provider folder ID (optional — avoids folder lookup)"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--items    "), descStyle.Render("Comma-separated item IDs (optional — load only specific items)"))
	fmt.Printf("    %s   %s\n", flagStyle.Render("--quiet    "), descStyle.Render("Suppress export messages for automatic hooks"))
	fmt.Println()

	fmt.Printf("  %s\n\n", headerStyle.Render("Quick Start:"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv init            # Interactive setup"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv refresh         # Sync provider and reload secrets"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv doctor          # Diagnose setup issues"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv status          # Check everything is working"))
	fmt.Printf("    %s\n", exampleStyle.Render("bwenv examples        # See all usage examples"))
	fmt.Println()

	fmt.Printf("  %s\n\n", descStyle.Render("Aliases: load → export, clean → remove, test → status, lock → logout, deny → disallow, settings → config, auth → login"))
}
