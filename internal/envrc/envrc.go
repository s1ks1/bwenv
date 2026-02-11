// Package envrc handles generating .envrc files and exporting environment
// variables from secret providers. It provides these capabilities:
//
//  1. Generate() — creates a .envrc file that calls "bwenv export" via direnv.
//  2. Export() — fetches secrets from a provider and prints "export KEY=VALUE"
//     lines to stdout, plus a rich boxed summary to stderr.
//  3. Remove() — deletes the .envrc file from the current directory.
//  4. AllowDirenv() — runs "direnv allow" to approve the .envrc automatically.
//  5. SilenceDirenvGlobally() — adds DIRENV_LOG_FORMAT="" to the user's shell
//     RC file so that ALL direnv messages are suppressed system-wide.
//
// The generated .envrc sets DIRENV_LOG_FORMAT="" and DIRENV_WARN_TIMEOUT="10m"
// as in-file defenses. However, the "direnv: loading .envrc" message is printed
// by direnv BEFORE the .envrc runs, so it can only be suppressed by having
// DIRENV_LOG_FORMAT="" already in the environment. SilenceDirenvGlobally()
// handles this by writing it to the user's shell RC (e.g. ~/.zshrc), which
// ensures zero direnv noise from the very first cd into a project directory.
package envrc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/provider"
)

// ── Styles for the export summary box (printed to stderr on every direnv load) ──

var (
	// boxBorder is the border style used for the export summary box.
	boxBorder = lipgloss.RoundedBorder()

	// summaryBrand is the "bwenv" label rendered above the box.
	summaryBrand = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"})

	// summaryMuted is used for secondary info (separators, hints, dim text).
	summaryMuted = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})

	// summarySuccess is the green style for success indicators and counts.
	summarySuccess = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"})

	// summaryVarName styles individual variable names inside the box.
	summaryVarName = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B21A8", Dark: "#C084FC"})

	// summaryContext styles the provider/folder line inside the box.
	summaryContext = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"})

	// summaryError is the red style for error messages inside the box.
	summaryError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"})

	// summaryBox is the bordered box that wraps the entire export summary.
	summaryBox = lipgloss.NewStyle().
			BorderStyle(boxBorder).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#0066CC", Dark: "#58A6FF"}).
			Padding(0, 1)

	// summaryBoxError is the bordered box for error summaries (red border).
	summaryBoxError = lipgloss.NewStyle().
			BorderStyle(boxBorder).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}).
			Padding(0, 1)
)

// ── Config ──────────────────────────────────────────────────────────────────

// Config holds all the parameters needed to generate a .envrc file.
// These are collected during the interactive init flow and written into
// the .envrc so that "bwenv export" can reconstruct the same context
// when direnv loads the file later.
type Config struct {
	// ProviderSlug is the short identifier for the chosen provider (e.g. "bitwarden").
	ProviderSlug string

	// FolderName is the human-readable folder name chosen by the user.
	FolderName string

	// FolderID is the provider-specific unique identifier for the folder.
	// Stored in the .envrc so we don't have to look it up by name every time.
	FolderID string

	// Session is the authentication token (e.g. BW_SESSION for Bitwarden).
	// For providers that manage sessions internally (like 1Password v2), this may be empty.
	Session string

	// Version is the bwenv version that generated this file (for debugging).
	Version string
}

// ── .envrc generation ───────────────────────────────────────────────────────

// Generate creates a .envrc file in the current directory. The generated file
// uses direnv's eval mechanism to call "bwenv export", which fetches secrets
// from the configured provider and folder at shell load time.
//
// Key design decisions in the generated .envrc:
//
//   - DIRENV_LOG_FORMAT is set to "" to suppress direnv's own export/unload
//     messages (e.g. "direnv: export +VAR1 +VAR2..."). This takes effect on
//     the SECOND load and onward (since direnv reads the value before running
//     the .envrc). For first-load suppression, see SilenceDirenvGlobally().
//   - DIRENV_WARN_TIMEOUT uses Go duration format ("10m") which is required
//     by direnv v2.30+ (plain seconds like "600" cause parse errors).
//   - BW_SESSION is stored for Bitwarden users (the token expires, so the
//     user will need to re-run "bwenv init" when it does).
func Generate(cfg Config) error {
	var b strings.Builder

	// -- Header with metadata --
	b.WriteString("# ═══════════════════════════════════════════════════════════════\n")
	b.WriteString(fmt.Sprintf("# Generated by bwenv %s on %s\n",
		cfg.Version, time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("# Provider: %s | Folder: %s\n", cfg.ProviderSlug, cfg.FolderName))
	b.WriteString("#\n")
	b.WriteString("# DO NOT edit manually — re-run 'bwenv init' to regenerate.\n")
	b.WriteString("# ═══════════════════════════════════════════════════════════════\n")
	b.WriteString("\n")

	// -- Silence direnv's log output --
	// Setting DIRENV_LOG_FORMAT="" suppresses direnv messages like:
	//   - "direnv: export +VAR1 +VAR2..."
	//   - "direnv: unloading"
	// The "direnv: loading .envrc" message is printed BEFORE the .envrc runs,
	// so it can only be silenced by having this variable already in the shell
	// environment. bwenv init handles that via SilenceDirenvGlobally().
	b.WriteString("# Silence direnv's own output — bwenv handles all user feedback\n")
	b.WriteString("export DIRENV_LOG_FORMAT=\"\"\n")
	b.WriteString("\n")

	// -- Set a generous timeout --
	// Vault operations (unlock, fetch secrets) can take several seconds.
	// direnv warns after 5s by default and that looks like an error to users.
	// We use Go duration format ("10m") which direnv v2.30+ requires.
	// Plain numbers like "600" cause: "invalid DIRENV_WARN_TIMEOUT: time: missing unit"
	b.WriteString("# Generous timeout for vault operations (Go duration format for direnv v2.30+)\n")
	b.WriteString("export DIRENV_WARN_TIMEOUT=\"10m\"\n")
	b.WriteString("\n")

	// -- Bitwarden session token (if applicable) --
	if cfg.Session != "" {
		b.WriteString("# Bitwarden session token (required for vault access)\n")
		b.WriteString("# This token expires — re-run 'bwenv init' if you get auth errors.\n")
		b.WriteString(fmt.Sprintf("export BW_SESSION=%s\n", shellQuote(cfg.Session)))
		b.WriteString("\n")
	}

	// -- The main payload --
	// eval runs "bwenv export" which prints "export KEY=VALUE" lines to stdout.
	// bwenv also prints a styled summary to stderr so the user sees what loaded.
	// Since bwenv's output does NOT start with "direnv:", it is never confused
	// with direnv's own messages.
	b.WriteString("# Load secrets from the provider into the environment\n")
	b.WriteString(fmt.Sprintf("eval \"$(bwenv export --provider %s --folder %s)\"\n",
		shellEscape(cfg.ProviderSlug),
		shellQuote(cfg.FolderName),
	))

	// Write the file with restrictive permissions (owner read/write only).
	// The .envrc may contain sensitive data like BW_SESSION tokens.
	if err := os.WriteFile(".envrc", []byte(b.String()), 0600); err != nil {
		return fmt.Errorf("failed to write .envrc: %w", err)
	}

	return nil
}

// ── direnv helpers ──────────────────────────────────────────────────────────

// AllowDirenv runs "direnv allow" in the current directory so the user
// doesn't have to do it manually after "bwenv init". This prevents the
// scary "direnv: error .envrc is blocked" message from appearing.
//
// If direnv is not installed or the allow command fails, we return an error
// but this should be treated as non-fatal — the user can always run it manually.
func AllowDirenv() error {
	// Check if direnv is available before trying to call it.
	if _, err := exec.LookPath("direnv"); err != nil {
		return fmt.Errorf("direnv not found in PATH")
	}

	cmd := exec.Command("direnv", "allow")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("'direnv allow' failed: %w", err)
	}

	return nil
}

// SilenceDirenvGlobally adds `export DIRENV_LOG_FORMAT=""` to the user's
// shell RC file (e.g. ~/.zshrc, ~/.bashrc). This ensures that ALL direnv
// messages are suppressed — including "direnv: loading .envrc" which is
// printed BEFORE the .envrc file runs and therefore cannot be silenced
// from within .envrc itself.
//
// This is a one-time operation that bwenv init calls after generating the
// .envrc. It is idempotent — if the line already exists in any RC file,
// it does nothing.
//
// Returns (modified bool, filePath string, err error):
//   - modified=true, filePath="~/.zshrc" → line was added to ~/.zshrc
//   - modified=false, filePath="~/.zshrc" → already present in ~/.zshrc
//   - modified=false, filePath="", err → could not determine shell RC
func SilenceDirenvGlobally() (modified bool, filePath string, err error) {
	// The magic line we need in the shell RC.
	const silenceLine = `export DIRENV_LOG_FORMAT=""`
	const marker = "DIRENV_LOG_FORMAT"

	// Determine which shell RC file to modify.
	rcPath, err := detectShellRC()
	if err != nil {
		return false, "", err
	}

	// Shorten for display purposes (e.g. /Users/john/.zshrc → ~/.zshrc).
	displayPath := shortenHomePath(rcPath)

	// Read the existing file content (if it exists).
	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, displayPath, fmt.Errorf("could not read %s: %w", displayPath, err)
	}

	// Check if the marker is already present.
	if strings.Contains(string(content), marker) {
		return false, displayPath, nil
	}

	// Append the silence line with a comment explaining why it's there.
	appendContent := "\n# Silence direnv messages — bwenv provides its own styled output\n"
	appendContent += silenceLine + "\n"

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, displayPath, fmt.Errorf("could not write to %s: %w", displayPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(appendContent); err != nil {
		return false, displayPath, fmt.Errorf("failed to append to %s: %w", displayPath, err)
	}

	return true, displayPath, nil
}

// detectShellRC returns the path to the user's primary shell RC file.
// It checks the SHELL environment variable and maps to the corresponding RC file.
func detectShellRC() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	// Detect the user's shell from the SHELL env var.
	shellPath := os.Getenv("SHELL")
	shellName := filepath.Base(shellPath)

	switch shellName {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		// On macOS, .bash_profile is preferred over .bashrc for login shells.
		if runtime.GOOS == "darwin" {
			profile := filepath.Join(home, ".bash_profile")
			if _, err := os.Stat(profile); err == nil {
				return profile, nil
			}
		}
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		// Fallback: try zshrc (default on macOS), then bashrc.
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".zshrc"), nil
		}
		return filepath.Join(home, ".bashrc"), nil
	}
}

// shortenHomePath replaces the user's home directory prefix with "~"
// for more compact and readable display in status messages.
func shortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// ── Export command ───────────────────────────────────────────────────────────

// Export fetches secrets from the specified provider and folder, then prints
// "export KEY=VALUE" lines to stdout. It also prints a rich, boxed summary
// to stderr showing which variables were loaded.
//
// This function is designed to be called from within an .envrc file via eval:
//
//	eval "$(bwenv export --provider bitwarden --folder MyFolder)"
//
// stdout: only "export KEY=VALUE" lines (consumed by eval)
// stderr: styled box summary for the user (visible in the terminal)
func Export(providerSlug string, folderName string) error {
	// Look up the requested provider from the registry.
	p, err := provider.Get(providerSlug)
	if err != nil {
		printExportError("Provider not found", err)
		return err
	}

	// Check that the provider's CLI tool is available on this system.
	if !p.IsAvailable() {
		err := fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
		printExportError(fmt.Sprintf("%s unavailable", p.Name()), err)
		return err
	}

	// Authenticate with the provider. For Bitwarden, this uses BW_SESSION
	// from the environment. For 1Password, this triggers system auth.
	session, err := p.Authenticate()
	if err != nil {
		printExportError("Authentication failed", err)
		return fmt.Errorf("authentication failed for %s: %w", p.Name(), err)
	}

	// Fetch the list of folders so we can find the one matching the given name.
	folders, err := p.ListFolders(session)
	if err != nil {
		printExportError("Could not list folders", err)
		return fmt.Errorf("failed to list folders from %s: %w", p.Name(), err)
	}

	// Find the folder by name (case-sensitive match).
	var targetFolder *provider.Folder
	for _, f := range folders {
		if f.Name == folderName {
			matched := f // Copy to avoid referencing the loop variable.
			targetFolder = &matched
			break
		}
	}

	if targetFolder == nil {
		available := make([]string, len(folders))
		for i, f := range folders {
			available[i] = f.Name
		}
		err := fmt.Errorf("folder %q not found — available: %s",
			folderName, strings.Join(available, ", "))
		printExportError("Folder not found", err)
		return err
	}

	// Fetch all secrets from the folder.
	secrets, err := p.GetSecrets(session, *targetFolder)
	if err != nil {
		printExportError("Could not fetch secrets", err)
		return fmt.Errorf("failed to get secrets from folder %q: %w", folderName, err)
	}

	// Collect variable names for the summary (before printing export lines).
	varNames := make([]string, 0, len(secrets))

	// Print each secret as an export statement to stdout.
	// direnv will eval this output to set the environment variables.
	for _, s := range secrets {
		key := sanitizeKey(s.Key)
		fmt.Printf("export %s=%s\n", key, shellQuote(s.Value))
		varNames = append(varNames, key)
	}

	// Print a rich, boxed summary to stderr so the user sees what happened.
	// This goes to stderr to avoid polluting the eval'd stdout.
	printExportSummary(p.Name(), folderName, varNames)

	return nil
}

// ── Secret preview ──────────────────────────────────────────────────────────

// PreviewSecrets fetches secrets from the given provider and folder and returns
// just the key names (not values). This is used during "bwenv init" to show
// the user what variables will be loaded, without exposing actual secret values.
func PreviewSecrets(p provider.Provider, session string, folder provider.Folder) ([]string, error) {
	secrets, err := p.GetSecrets(session, folder)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(secrets))
	for _, s := range secrets {
		names = append(names, sanitizeKey(s.Key))
	}
	return names, nil
}

// ── Remove ──────────────────────────────────────────────────────────────────

// Remove deletes the .envrc file in the current directory.
// Returns (true, nil) if the file was removed, (false, nil) if it didn't exist,
// or (false, error) if deletion failed.
func Remove() (bool, error) {
	_, err := os.Stat(".envrc")
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not check .envrc: %w", err)
	}

	if err := os.Remove(".envrc"); err != nil {
		return false, fmt.Errorf("failed to remove .envrc: %w", err)
	}

	return true, nil
}

// ── Export summary output (printed to stderr) ──────────────────────────────

// printExportSummary prints a rich, boxed summary of what was loaded.
// This is called at the end of Export() and appears in the user's terminal
// every time direnv loads the .envrc (i.e., when they cd into the directory).
//
// The output is a compact bordered box that shows the provider, folder,
// variable count, and each variable name with a key icon — all styled with
// Lipgloss so it looks great on every terminal.
//
// Example output:
//
//	 🔐 bwenv
//	╭──────────────────────────────────────────╮
//	│  Bitwarden / MySecrets                   │
//	│                                          │
//	│  ✓ 3 variable(s) loaded                  │
//	│    🔑 DB_USERNAME                         │
//	│    🔑 DB_PASSWORD                         │
//	│    🔑 API_TOKEN                           │
//	╰──────────────────────────────────────────╯
func printExportSummary(providerName string, folderName string, varNames []string) {
	var lines []string

	// Line 1: Provider / Folder context.
	contextLine := summaryContext.Render(fmt.Sprintf("%s / %s", providerName, folderName))
	lines = append(lines, contextLine)

	// Empty separator line.
	lines = append(lines, "")

	if len(varNames) == 0 {
		// No variables found — show a warning.
		warningLine := summaryError.Render("⚠  No variables found in this folder")
		lines = append(lines, warningLine)
	} else {
		// Success line with count.
		countLine := summarySuccess.Render(fmt.Sprintf("✓ %d variable(s) loaded", len(varNames)))
		lines = append(lines, countLine)

		// List each variable name with a key icon.
		// If there are many variables, show the first batch and summarize the rest.
		const maxShown = 12
		shown := varNames
		truncated := false
		if len(shown) > maxShown {
			shown = shown[:maxShown]
			truncated = true
		}

		for _, name := range shown {
			varLine := fmt.Sprintf("  🔑 %s", summaryVarName.Render(name))
			lines = append(lines, varLine)
		}

		if truncated {
			remaining := len(varNames) - maxShown
			moreLine := summaryMuted.Render(fmt.Sprintf("  ... and %d more", remaining))
			lines = append(lines, moreLine)
		}
	}

	// Compose the box content and render it.
	content := strings.Join(lines, "\n")
	box := summaryBox.Render(content)

	// Print a header line above the box with the bwenv branding.
	brand := summaryBrand.Render("🔐 bwenv")
	fmt.Fprintf(os.Stderr, "\n %s\n%s\n", brand, box)
}

// printExportError prints a compact boxed error to stderr during export.
// This replaces the raw error message that would otherwise confuse users
// when direnv loads the .envrc and something goes wrong.
func printExportError(label string, err error) {
	var lines []string

	errorLabel := summaryError.Render("✗ " + label)
	lines = append(lines, errorLabel)
	lines = append(lines, "")

	detail := summaryMuted.Render(err.Error())
	lines = append(lines, detail)

	// Compose the error box and render it.
	content := strings.Join(lines, "\n")
	box := summaryBoxError.Render(content)

	brand := summaryBrand.Render("🔐 bwenv")
	fmt.Fprintf(os.Stderr, "\n %s\n%s\n", brand, box)
}

// ── Shell escaping helpers ─────────────────────────────────────────────────

// sanitizeKey ensures an environment variable name is valid for POSIX shells.
// It replaces any characters that aren't alphanumeric or underscores with
// underscores. Leading digits are prefixed with an underscore since env var
// names can't start with a digit.
func sanitizeKey(key string) string {
	if key == "" {
		return "_EMPTY_KEY"
	}

	var b strings.Builder
	for i, ch := range key {
		switch {
		case ch >= 'A' && ch <= 'Z':
			b.WriteRune(ch)
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			if i == 0 {
				b.WriteRune('_') // Env var names can't start with a digit.
			}
			b.WriteRune(ch)
		case ch == '_':
			b.WriteRune(ch)
		default:
			b.WriteRune('_') // Replace any special character with underscore.
		}
	}

	result := b.String()
	if result == "" {
		return "_EMPTY_KEY"
	}
	return result
}

// shellQuote wraps a value in single quotes for safe use in shell export
// statements. Single quotes in the value itself are escaped using the
// standard POSIX shell trick: end the quoted string, add an escaped
// single quote, then start a new quoted string.
//
// Example: it's → 'it'\”s'
func shellQuote(value string) string {
	escaped := strings.ReplaceAll(value, "'", "'\\''")
	return "'" + escaped + "'"
}

// shellEscape strips shell metacharacters from identifiers like provider slugs.
// Only alphanumeric characters, hyphens, underscores, and dots are kept.
func shellEscape(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
