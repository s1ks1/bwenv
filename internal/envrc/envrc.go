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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/internal/config"
	"github.com/s1ks1/bwenv/internal/provider"
)

// emojiStr returns the emoji if ShowEmoji is enabled in the user config,
// otherwise returns the plain-text fallback. Convenience wrapper for use
// within the envrc package so we don't import the ui package (which would
// create a circular dependency).
func emojiStr(emoji string, fallback string) string {
	return config.Emoji(emoji, fallback)
}

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
	// Load user preferences to decide whether to silence direnv output.
	userCfg, _ := config.Load()

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

	// -- Silence direnv's log output (unless user opted to see it) --
	// Setting DIRENV_LOG_FORMAT="" suppresses direnv messages like:
	//   - "direnv: export +VAR1 +VAR2..."
	//   - "direnv: unloading"
	// The "direnv: loading .envrc" message is printed BEFORE the .envrc runs,
	// so it can only be silenced by having this variable already in the shell
	// environment. bwenv init handles that via SilenceDirenvGlobally().
	if !userCfg.ShowDirenvOutput {
		b.WriteString("# Replace direnv's noisy output with a subtle, styled message\n")
		b.WriteString("export DIRENV_LOG_FORMAT=$'\\033[2m  \\U0001f510 %s\\033[0m'\n")
		b.WriteString("\n")
	} else {
		b.WriteString("# Direnv output is visible (configured via 'bwenv config')\n")
		b.WriteString("\n")
	}

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
		b.WriteString("# This token expires — run 'bwenv login' to re-authenticate.\n")
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

// UpdateSession replaces the BW_SESSION token in the existing .envrc file
// with a fresh one. This is called by "bwenv login" (TTY mode) and
// "bwenv allow" (TTY mode) after re-authenticating, so that subsequent
// direnv loads use the new token instead of the stale one.
//
// If the .envrc doesn't contain a BW_SESSION line (e.g. 1Password provider),
// this is a no-op. If the .envrc doesn't exist, it returns an error.
func UpdateSession(newSession string) error {
	if newSession == "" {
		return nil // Nothing to update (e.g. 1Password doesn't use session tokens).
	}

	content, err := os.ReadFile(".envrc")
	if err != nil {
		return fmt.Errorf("could not read .envrc: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false
	newLine := fmt.Sprintf("export BW_SESSION=%s", shellQuote(newSession))

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "export BW_SESSION=") {
			lines[i] = newLine
			found = true
			break
		}
	}

	if !found {
		// No BW_SESSION line exists — nothing to update.
		return nil
	}

	if err := os.WriteFile(".envrc", []byte(strings.Join(lines, "\n")), 0600); err != nil {
		return fmt.Errorf("failed to update .envrc: %w", err)
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
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard // Suppress "direnv:" messages
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("'direnv allow' failed: %w", err)
	}

	return nil
}

// AllowAndExport is the handler for `eval "$(bwenv allow)"`. It:
//  1. Parses .envrc to get provider/folder info.
//  2. Authenticates (may prompt for password ONCE).
//  3. Fetches secrets and prints export lines to stdout.
//  4. Also exports BW_SESSION (if applicable) and DIRENV_LOG_FORMAT=""
//     so that when direnv's hook re-fires after eval completes, the
//     subshell inherits a valid session and stays silent — no second
//     password prompt.
//  5. Runs "direnv allow" LAST so the hook fires only after the shell
//     already has all the right env vars.
func AllowAndExport() (providerSlug string, folderName string, err error) {
	// Step 1: Parse .envrc to get provider/folder info.
	providerSlug, folderName, err = ParseEnvrcConfig()
	if err != nil {
		return "", "", fmt.Errorf("could not parse .envrc: %w", err)
	}

	// Step 2: Authenticate and export secrets. This is the one-and-only
	// place the user may be prompted for their master password.
	session, exportErr := ExportInteractive(providerSlug, folderName)
	if exportErr != nil {
		return providerSlug, folderName, fmt.Errorf("export failed: %w", exportErr)
	}

	// Step 3: Output the fresh session token so the parent shell has it.
	// When direnv's hook re-fires .envrc, the subshell will inherit this
	// fresh BW_SESSION from the parent environment, overriding the
	// potentially stale one in .envrc. No second password prompt.
	if session != "" {
		fmt.Printf("export BW_SESSION=%s\n", shellQuote(session))
	}

	// Step 4: Ensure DIRENV_LOG_FORMAT is set in the parent shell so the
	// hook re-fire uses our styled format instead of the ugly default.
	fmt.Printf("export DIRENV_LOG_FORMAT=$'\\033[2m  \\U0001f510 %%s\\033[0m'\n")
	fmt.Printf("export DIRENV_WARN_TIMEOUT=\"10m\"\n")

	// Step 5: Run direnv allow LAST. The shell now has fresh BW_SESSION
	// and DIRENV_LOG_FORMAT, so when the hook fires the re-load is both
	// silent and auth-free.
	if allowErr := AllowDirenv(); allowErr != nil {
		// Non-fatal — direnv may not be installed.
		_ = allowErr
	}

	return providerSlug, folderName, nil
}

// LoginAndExport is the handler for `eval "$(bwenv login)"`. It:
//  1. Parses .envrc to get provider/folder info.
//  2. Authenticates interactively (may prompt for master password).
//  3. Fetches secrets and prints export lines to stdout.
//  4. Exports the fresh BW_SESSION (if applicable) so the parent shell
//     inherits a valid token — direnv re-fires silently.
//  5. Runs "direnv allow" so the .envrc is trusted.
//
// This is functionally identical to AllowAndExport but semantically different:
// it's the recovery path when a session expires, while AllowAndExport is the
// initial approval path. Having a distinct "login" command makes the UX clearer.
func LoginAndExport() (providerSlug string, folderName string, err error) {
	// Step 1: Parse .envrc to get provider/folder info.
	providerSlug, folderName, err = ParseEnvrcConfig()
	if err != nil {
		return "", "", fmt.Errorf("could not parse .envrc: %w", err)
	}

	// Step 2: Authenticate and export secrets. This is the one-and-only
	// place the user may be prompted for their master password.
	session, exportErr := ExportInteractive(providerSlug, folderName)
	if exportErr != nil {
		return providerSlug, folderName, fmt.Errorf("export failed: %w", exportErr)
	}

	// Step 3: Output the fresh session token so the parent shell has it.
	if session != "" {
		fmt.Printf("export BW_SESSION=%s\n", shellQuote(session))
	}

	// Step 4: Ensure DIRENV_LOG_FORMAT is set in the parent shell.
	fmt.Printf("export DIRENV_LOG_FORMAT=$'\\033[2m  \\U0001f510 %%s\\033[0m'\n")
	fmt.Printf("export DIRENV_WARN_TIMEOUT=\"10m\"\n")

	// Step 5: Run direnv allow LAST.
	if allowErr := AllowDirenv(); allowErr != nil {
		_ = allowErr // Non-fatal.
	}

	return providerSlug, folderName, nil
}

// ReauthenticateProvider authenticates with the named provider and returns
// the fresh session token. This is a lightweight helper for callers that
// need to refresh the session without exporting secrets (e.g. "bwenv allow"
// in TTY mode, which just needs to update the BW_SESSION in .envrc).
//
// Returns ("", nil) for providers that don't use session tokens (e.g. 1Password).
func ReauthenticateProvider(providerSlug string) (string, error) {
	p, err := provider.Get(providerSlug)
	if err != nil {
		return "", fmt.Errorf("provider %q not found: %w", providerSlug, err)
	}

	if !p.IsAvailable() {
		return "", fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
	}

	session, err := p.Authenticate()
	if err != nil {
		return "", fmt.Errorf("authentication failed for %s: %w", p.Name(), err)
	}

	return session, nil
}

// ParseEnvrcConfig reads the .envrc in the current directory and extracts
// the provider slug and folder name. Returns an error if the file doesn't
// exist or doesn't appear to be generated by bwenv.
func ParseEnvrcConfig() (providerSlug string, folderName string, err error) {
	content, err := os.ReadFile(".envrc")
	if err != nil {
		return "", "", fmt.Errorf("no .envrc found in current directory")
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "bwenv") {
		return "", "", fmt.Errorf(".envrc was not generated by bwenv")
	}

	// Try to extract from the header comment first:
	// Format: "# Provider: bitwarden | Folder: MySecrets"
	for _, line := range strings.Split(contentStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# Provider:") {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) >= 1 {
				provPart := strings.TrimPrefix(parts[0], "# Provider:")
				providerSlug = strings.TrimSpace(provPart)
			}
			if len(parts) >= 2 {
				folderPart := strings.TrimPrefix(parts[1], " Folder:")
				folderName = strings.TrimSpace(folderPart)
			}
			break
		}
	}

	// Fallback: extract from the bwenv export command line.
	if providerSlug == "" || folderName == "" {
		for _, line := range strings.Split(contentStr, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "bwenv export") {
				if idx := strings.Index(line, "--provider"); idx >= 0 {
					rest := line[idx+len("--provider"):]
					rest = strings.TrimSpace(rest)
					fields := strings.Fields(rest)
					if len(fields) > 0 {
						providerSlug = fields[0]
					}
				}
				if idx := strings.Index(line, "--folder"); idx >= 0 {
					rest := line[idx+len("--folder"):]
					rest = strings.TrimSpace(rest)
					// Folder may be quoted.
					if len(rest) > 0 && (rest[0] == '\'' || rest[0] == '"') {
						quote := rest[0]
						end := strings.IndexByte(rest[1:], quote)
						if end >= 0 {
							folderName = rest[1 : end+1]
						}
					} else {
						fields := strings.Fields(rest)
						if len(fields) > 0 {
							folderName = strings.TrimSuffix(fields[0], ")")
							folderName = strings.Trim(folderName, "'\"")
						}
					}
				}
				break
			}
		}
	}

	if providerSlug == "" || folderName == "" {
		return providerSlug, folderName, fmt.Errorf("could not extract provider and folder from .envrc")
	}

	return providerSlug, folderName, nil
}

// DisallowDirenv runs "direnv deny" in the current directory to block
// the .envrc file from being loaded. This is the inverse of AllowDirenv.
func DisallowDirenv() error {
	// Check if direnv is available before trying to call it.
	if _, err := exec.LookPath("direnv"); err != nil {
		return fmt.Errorf("direnv not found in PATH")
	}

	cmd := exec.Command("direnv", "deny")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard // Suppress "direnv:" messages
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("'direnv deny' failed: %w", err)
	}

	return nil
}

// DisallowAndUnset blocks the .envrc via direnv deny AND prints "unset VAR"
// statements to stdout for every variable that bwenv exported.
// Uses the .bwenv_vars cache to know which secret variables to unset.
// DIRENV_LOG_FORMAT and DIRENV_WARN_TIMEOUT are intentionally NOT unset
// so direnv stays quiet.
func DisallowAndUnset() ([]string, error) {
	varNames := loadCachedVarNames()

	if err := DisallowDirenv(); err != nil {
		return varNames, err
	}

	// Print unset statements to stdout (captured by shell wrapper's eval).
	for _, name := range varNames {
		fmt.Printf("unset %s\n", name)
	}

	return varNames, nil
}

// RemoveAndUnset removes .envrc and .bwenv_vars, calls direnv deny,
// AND prints "unset VAR" statements to stdout.
func RemoveAndUnset() (bool, []string, error) {
	// Load cached variable names BEFORE deleting any files.
	varNames := loadCachedVarNames()

	removed, _, err := Remove()
	if err != nil {
		return removed, varNames, err
	}
	if !removed {
		return false, nil, nil
	}

	// Print unset statements to stdout (captured by shell wrapper's eval).
	for _, name := range varNames {
		fmt.Printf("unset %s\n", name)
	}

	return true, varNames, nil
}

// ── Variable name cache ─────────────────────────────────────────────────────

// bwenvVarsCacheFile is the file where bwenv export saves the variable names
// it exported. This allows disallow/remove to know which vars to unset without
// re-authenticating with the provider.
const bwenvVarsCacheFile = ".bwenv_vars"

// saveVarNamesCache writes the exported variable names to .bwenv_vars.
// Called by exportSecrets() after every successful export.
func saveVarNamesCache(varNames []string) {
	if len(varNames) == 0 {
		return
	}
	_ = os.WriteFile(bwenvVarsCacheFile, []byte(strings.Join(varNames, "\n")+"\n"), 0600)
}

// loadCachedVarNames reads variable names from .bwenv_vars (written by export).
// Falls back to parsing .envrc static exports if the cache doesn't exist.
func loadCachedVarNames() []string {
	// Primary: read from cache file (has the actual secret var names).
	if content, err := os.ReadFile(bwenvVarsCacheFile); err == nil {
		var names []string
		for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
			if line != "" {
				names = append(names, line)
			}
		}
		if len(names) > 0 {
			// Also add BW_SESSION to unset so stale tokens don't linger.
			names = append(names, "BW_SESSION")
			return names
		}
	}

	// Fallback: parse static exports from .envrc.
	return parseEnvrcVarNames()
}

// removeCachedVarNames deletes the .bwenv_vars cache file.
func removeCachedVarNames() {
	_ = os.Remove(bwenvVarsCacheFile)
}

// parseEnvrcVarNames reads the .envrc and extracts variable names from
// "export KEY=..." lines. Variables managed by direnv internally
// (DIRENV_LOG_FORMAT, DIRENV_WARN_TIMEOUT) are excluded because we
// want those to stay set so direnv remains silent.
func parseEnvrcVarNames() []string {
	content, err := os.ReadFile(".envrc")
	if err != nil {
		return nil
	}

	// These are direnv control variables — never unset them.
	skip := map[string]bool{
		"DIRENV_LOG_FORMAT":   true,
		"DIRENV_WARN_TIMEOUT": true,
	}

	var names []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			rest := strings.TrimPrefix(line, "export ")
			if idx := strings.Index(rest, "="); idx > 0 {
				key := rest[:idx]
				if !seen[key] && !skip[key] {
					seen[key] = true
					names = append(names, key)
				}
			}
		}
	}

	return names
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
	// The styled format replaces direnv's default "direnv: ..." messages with
	// a dimmed, bwenv-branded line: "  \U0001f510 loading .envrc" (dimmed).
	// Using $'...' syntax for ANSI escape and unicode support.
	const silenceLine = `export DIRENV_LOG_FORMAT=$'\033[2m  \U0001f510 %s\033[0m'`
	const timeoutLine = `export DIRENV_WARN_TIMEOUT="10m"`
	const markerLog = "DIRENV_LOG_FORMAT"
	const markerTimeout = "DIRENV_WARN_TIMEOUT"

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

	contentStr := string(content)
	hasLog := strings.Contains(contentStr, markerLog)
	hasTimeout := strings.Contains(contentStr, markerTimeout)

	// If both markers are already present, nothing to do.
	if hasLog && hasTimeout {
		return false, displayPath, nil
	}

	// Build the content to append — only add lines that are missing.
	var appendContent string
	if !hasLog || !hasTimeout {
		appendContent += "\n# Silence direnv messages — bwenv provides its own styled output\n"
	}
	if !hasLog {
		appendContent += silenceLine + "\n"
	}
	if !hasTimeout {
		appendContent += timeoutLine + "\n"
	}

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

// ── Shell wrapper ───────────────────────────────────────────────────────────

// shellWrapperMarker is the unique string we look for to detect if the
// bwenv shell wrapper function is already installed.
const shellWrapperMarker = "# bwenv shell integration"

// shellWrapperBashZsh is the shell function for bash/zsh that wraps bwenv
// commands. Commands that produce shell code (export/unset) are eval'd
// transparently, so "bwenv allow" / "bwenv disallow" / "bwenv remove" /
// "bwenv login" can modify the current shell's environment directly.
const shellWrapperBashZsh = `
# bwenv shell integration — enables seamless secret management
# Commands like allow/disallow/remove/login modify your shell environment directly.
bwenv() {
  case "${1:-}" in
    allow|disallow|deny|remove|clean|export|load|login|auth)
      local _bwenv_out
      _bwenv_out="$(command bwenv "$@")"
      local _bwenv_rc=$?
      [ $_bwenv_rc -eq 0 ] && [ -n "$_bwenv_out" ] && eval "$_bwenv_out"
      return $_bwenv_rc
      ;;
    *)
      command bwenv "$@"
      ;;
  esac
}
`

// shellWrapperFish is the shell function for fish shell.
const shellWrapperFish = `
# bwenv shell integration — enables seamless secret management
function bwenv
  switch $argv[1]
    case allow disallow deny remove clean export load login auth
      set -l _out (command bwenv $argv)
      set -l _rc $status
      if test $_rc -eq 0 -a -n "$_out"
        eval $_out
      end
      return $_rc
    case '*'
      command bwenv $argv
  end
end
`

// InstallShellWrapper adds the bwenv() shell wrapper function to the user's
// shell RC file. This wrapper transparently eval's the output of commands
// like "bwenv allow", "bwenv disallow", and "bwenv remove" so they can
// modify the current shell's environment (set/unset variables) directly.
//
// Without the wrapper, these commands would require the user to manually
// type eval "$(bwenv allow)" etc.
//
// Returns (modified bool, filePath string, err error):
//   - modified=true  → wrapper was added to the RC file
//   - modified=false → wrapper already present or error occurred
func InstallShellWrapper() (modified bool, filePath string, err error) {
	rcPath, err := detectShellRC()
	if err != nil {
		return false, "", err
	}

	displayPath := shortenHomePath(rcPath)

	// Read the existing file to check if wrapper is already installed.
	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, displayPath, fmt.Errorf("could not read %s: %w", displayPath, err)
	}

	if strings.Contains(string(content), shellWrapperMarker) {
		return false, displayPath, nil
	}

	// Determine which wrapper to install based on the shell.
	shellName := filepath.Base(os.Getenv("SHELL"))
	wrapper := shellWrapperBashZsh
	if shellName == "fish" {
		wrapper = shellWrapperFish
	}

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, displayPath, fmt.Errorf("could not write to %s: %w", displayPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(wrapper); err != nil {
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

// shortenHomePath uses the shared ui helper (kept as a local alias
// to avoid importing the ui package which would create circular deps).
func shortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
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
// This function is called by direnv inside .envrc via eval. It NEVER prompts
// for a password interactively — the session must already be available via
// BW_SESSION env var (set by the .envrc itself or inherited from the parent
// shell). If the session is invalid, it fails with a clear error message
// telling the user to re-run "bwenv init".
//
// stdout: only "export KEY=VALUE" lines (consumed by eval)
// stderr: styled box summary for the user (visible in the terminal)
func Export(providerSlug string, folderName string) error {
	_, err := exportSecrets(providerSlug, folderName, false)
	return err
}

// ExportInteractive is like Export but allows interactive authentication
// (prompting for a master password). This is used by "bwenv allow" where
// the user explicitly runs bwenv and expects to enter their password once.
// Returns the session token so the caller can propagate it.
func ExportInteractive(providerSlug string, folderName string) (string, error) {
	return exportSecrets(providerSlug, folderName, true)
}

// exportSecrets is the shared implementation for Export and ExportInteractive.
// When interactive=false (direnv context), authentication failures produce a
// helpful error instead of blocking on a password prompt.
func exportSecrets(providerSlug string, folderName string, interactive bool) (string, error) {
	// Load user preferences to decide whether to show the export summary.
	userCfg, _ := config.Load()

	// Look up the requested provider from the registry.
	p, err := provider.Get(providerSlug)
	if err != nil {
		printExportError("Provider not found", err)
		return "", err
	}

	// Check that the provider's CLI tool is available on this system.
	if !p.IsAvailable() {
		err := fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
		printExportError(fmt.Sprintf("%s unavailable", p.Name()), err)
		return "", err
	}

	// Authenticate with the provider.
	var session string
	if interactive {
		// Interactive mode (bwenv allow): may prompt for master password.
		session, err = p.Authenticate()
	} else {
		// Non-interactive mode (direnv .envrc): use existing session only.
		if !p.IsAuthenticated() {
			err = fmt.Errorf(
				"session expired or not active — run 'bwenv login' to re-authenticate")
			printExportError("Authentication required", err)
			return "", err
		}
		// Session is valid — get it from the provider.
		session, err = p.Authenticate()
	}
	if err != nil {
		printExportError("Authentication failed", err)
		return "", fmt.Errorf("authentication failed for %s: %w", p.Name(), err)
	}

	// Fetch the list of folders so we can find the one matching the given name.
	folders, err := p.ListFolders(session)
	if err != nil {
		printExportError("Could not list folders", err)
		return session, fmt.Errorf("failed to list folders from %s: %w", p.Name(), err)
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
		return session, err
	}

	// Fetch all secrets from the folder.
	secrets, err := p.GetSecrets(session, *targetFolder)
	if err != nil {
		printExportError("Could not fetch secrets", err)
		return session, fmt.Errorf("failed to get secrets from folder %q: %w", folderName, err)
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

	// Cache variable names so disallow/remove can unset them later
	// without needing to re-authenticate with the provider.
	saveVarNamesCache(varNames)

	// Print a rich, boxed summary to stderr so the user sees what happened.
	// This goes to stderr to avoid polluting the eval'd stdout.
	// Controlled by the ShowExportSummary config preference.
	if userCfg.ShowExportSummary {
		printExportSummary(p.Name(), folderName, varNames)
	}

	return session, nil
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
// Before deleting, it calls "direnv deny" so the direnv cache is invalidated
// and extracts the variable names that were exported for the caller to
// display unset hints.
// Returns (removed bool, varNames []string, err error).
func Remove() (bool, []string, error) {
	_, err := os.Stat(".envrc")
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("could not check .envrc: %w", err)
	}

	// Capture variable names before we delete the file.
	varNames := loadCachedVarNames()

	// Deny direnv so the cached allowance is revoked.
	// Non-fatal: direnv may not be installed when just cleaning up.
	if denyErr := DisallowDirenv(); denyErr != nil {
		// If direnv isn't installed, that's fine — just skip.
		if _, lookErr := exec.LookPath("direnv"); lookErr == nil {
			// direnv IS installed but deny failed — still not fatal.
			_ = denyErr
		}
	}

	if err := os.Remove(".envrc"); err != nil {
		return false, varNames, fmt.Errorf("failed to remove .envrc: %w", err)
	}

	// Also clean up the variable name cache.
	removeCachedVarNames()

	return true, varNames, nil
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
		warningLine := summaryError.Render(emojiStr("⚠ ", "!") + " No variables found in this folder")
		lines = append(lines, warningLine)
	} else {
		// Success line with count.
		countLine := summarySuccess.Render(fmt.Sprintf("%s %d variable(s) loaded", emojiStr("✓", "[OK]"), len(varNames)))
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
			varLine := fmt.Sprintf("  %s %s", emojiStr("🔑", " *"), summaryVarName.Render(name))
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
	brand := summaryBrand.Render(emojiStr("🔐", "[*]") + " bwenv")
	fmt.Fprintf(os.Stderr, "\n %s\n%s\n", brand, box)
}

// printExportError prints a compact boxed error to stderr during export.
// This replaces the raw error message that would otherwise confuse users
// when direnv loads the .envrc and something goes wrong.
func printExportError(label string, err error) {
	var lines []string

	errorLabel := summaryError.Render(emojiStr("✗", "X") + " " + label)
	lines = append(lines, errorLabel)
	lines = append(lines, "")

	detail := summaryMuted.Render(err.Error())
	lines = append(lines, detail)

	// Compose the error box and render it.
	content := strings.Join(lines, "\n")
	box := summaryBoxError.Render(content)

	brand := summaryBrand.Render(emojiStr("🔐", "[*]") + " bwenv")
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
