// Package export fetches secrets from a provider and prints shell export
// statements for the activation backend to eval, plus the allow/login/refresh
// orchestration and the variable-name cache used to unset secrets later.
//
// It never prompts during the direnv hot path: authentication is validated by
// the requested provider operation. .envrc generation and direnv control live
// in internal/activation/direnv.
package export

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/config"
	"github.com/s1ks1/bwenv/v3/internal/project"
	"github.com/s1ks1/bwenv/v3/internal/provider"
	"github.com/s1ks1/bwenv/v3/internal/shell"
)

// emojiStr returns the emoji if ShowEmoji is enabled in the user config,
// otherwise returns the plain-text fallback. Convenience wrapper for use
// within the envrc package so we don't import the ui package (which would
// create a circular dependency).
func emojiStr(emoji string, fallback string) string {
	return config.Emoji(emoji, fallback)
}

// activatorFor resolves the activation backend selected by the project's
// activation.mode. A legacy project without .bwenv.toml defaults to direnv.
// Core secret retrieval depends only on this interface, never on a concrete
// backend.
func activatorFor() (activation.Activator, error) {
	mode := "direnv"
	cfg, err := project.Load(".bwenv.toml")
	switch {
	case err == nil:
		mode = cfg.Activation.Mode
	case errors.Is(err, os.ErrNotExist):
		// Legacy .envrc project — direnv is the stable default backend.
	default:
		return nil, err
	}
	return activation.Get(mode)
}

// Activate prepares the nearest project's activation artifact and reports the
// backend used. It works from any nested subdirectory and is idempotent:
// activating an already-active project succeeds without duplicating state.
func Activate() (backend string, err error) {
	root, err := project.FindRoot(".")
	if err != nil {
		return "", err
	}
	if root == "" {
		return "", fmt.Errorf("no bwenv project found in this directory or any parent")
	}
	if err := os.Chdir(root); err != nil {
		return "", fmt.Errorf("enter project %s: %w", root, err)
	}

	activator, err := activatorFor()
	if err != nil {
		return "", err
	}

	// Backends that activate by emitting exports load the secrets now.
	if emitter, ok := activator.(activation.Emitter); ok && emitter.EmitsExports() {
		source, err := activator.Resolve()
		if err != nil {
			return "", err
		}
		if err := ExportWithFolderID(source.ProviderSlug, source.FolderName, source.FolderID, source.ItemIDs); err != nil {
			return "", err
		}
		return activator.Name(), nil
	}

	if !activator.Available() {
		return "", fmt.Errorf("%s backend is not installed", activator.Name())
	}
	if err := activator.Approve(); err != nil {
		return "", err
	}
	return activator.Name(), nil
}

// Deactivate revokes the nearest project's activation and prints "unset VAR"
// statements so the caller's shell restores its previous environment. It is
// idempotent: deactivating an inactive project clears nothing and succeeds.
func Deactivate() ([]string, error) {
	root, err := project.FindRoot(".")
	if err != nil {
		return nil, err
	}
	if root == "" {
		return nil, fmt.Errorf("no bwenv project found in this directory or any parent")
	}
	if err := os.Chdir(root); err != nil {
		return nil, fmt.Errorf("enter project %s: %w", root, err)
	}
	return DisallowAndUnset()
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
	// Step 1: Resolve the project's activation backend and its secret source.
	activator, err := activatorFor()
	if err != nil {
		return "", "", err
	}
	source, err := activator.Resolve()
	if err != nil {
		return "", "", fmt.Errorf("could not resolve the project source: %w", err)
	}
	providerSlug, folderName = source.ProviderSlug, source.FolderName

	// Step 2: Authenticate and export secrets. This is the one-and-only
	// place the user may be prompted for their master password.
	session, exportErr := ExportInteractive(providerSlug, folderName, nil)
	if exportErr != nil {
		return providerSlug, folderName, fmt.Errorf("export failed: %w", exportErr)
	}

	// Step 3: Output the fresh session token so the parent shell has it.
	// When direnv's hook re-fires .envrc, the subshell will inherit this
	// fresh BW_SESSION from the parent environment, overriding the
	// potentially stale one in .envrc. No second password prompt.
	if session != "" {
		fmt.Printf("export BW_SESSION=%s\n", shell.Quote(session))
	}

	// Step 4: Ensure DIRENV_LOG_FORMAT is set in the parent shell so the
	// hook re-fire uses our styled format instead of the ugly default.
	fmt.Printf("export DIRENV_LOG_FORMAT=$'\\033[2m  \\U0001f510 %%s\\033[0m'\n")
	fmt.Printf("export DIRENV_WARN_TIMEOUT=\"10m\"\n")

	// Step 5: Approve the activation artifact LAST. The shell now has a fresh
	// session, so the backend hook re-load is both silent and auth-free.
	if approveErr := activator.Approve(); approveErr != nil {
		// Non-fatal — the backend tooling may not be installed.
		_ = approveErr
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
	// Step 1: Resolve the project's activation backend and its secret source.
	activator, err := activatorFor()
	if err != nil {
		return "", "", err
	}
	source, err := activator.Resolve()
	if err != nil {
		return "", "", fmt.Errorf("could not resolve the project source: %w", err)
	}
	providerSlug, folderName = source.ProviderSlug, source.FolderName

	// Step 2: Authenticate and export secrets. This is the one-and-only
	// place the user may be prompted for their master password.
	session, exportErr := ExportInteractive(providerSlug, folderName, nil)
	if exportErr != nil {
		return providerSlug, folderName, fmt.Errorf("export failed: %w", exportErr)
	}

	// Step 3: Output the fresh session token so the parent shell has it.
	if session != "" {
		fmt.Printf("export BW_SESSION=%s\n", shell.Quote(session))
	}

	// Step 4: Ensure DIRENV_LOG_FORMAT is set in the parent shell.
	fmt.Printf("export DIRENV_LOG_FORMAT=$'\\033[2m  \\U0001f510 %%s\\033[0m'\n")
	fmt.Printf("export DIRENV_WARN_TIMEOUT=\"10m\"\n")

	// Step 5: Approve the activation artifact LAST.
	if approveErr := activator.Approve(); approveErr != nil {
		_ = approveErr // Non-fatal.
	}

	return providerSlug, folderName, nil
}

// Refresh syncs providers that support it and asks the activation backend to
// reload the current project's environment. Secret values are not written.
func Refresh() (providerName string, synced bool, err error) {
	activator, err := activatorFor()
	if err != nil {
		return "", false, err
	}
	source, err := activator.Resolve()
	if err != nil {
		return "", false, fmt.Errorf("could not read project configuration: %w", err)
	}
	p, err := provider.Get(source.ProviderSlug)
	if err != nil {
		return "", false, fmt.Errorf("could not determine the configured provider")
	}
	if !p.IsAvailable() {
		return "", false, fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
	}
	if !p.IsAuthenticated() {
		return "", false, fmt.Errorf("%s session is not active; run 'bwenv login'", p.Name())
	}
	if syncer, ok := p.(provider.Syncer); ok {
		if err := syncer.Sync(); err != nil {
			return "", false, err
		}
		synced = true
	}
	if err := activator.Reload(); err != nil {
		return p.Name(), synced, err
	}
	return p.Name(), synced, nil
}

// DisallowAndUnset blocks the .envrc via direnv deny AND prints "unset VAR"
// statements to stdout for every variable that bwenv exported.
// Uses the .bwenv_vars cache to know which secret variables to unset.
// DIRENV_LOG_FORMAT and DIRENV_WARN_TIMEOUT are intentionally NOT unset
// so direnv stays quiet.
func DisallowAndUnset() ([]string, error) {
	varNames := loadCachedVarNames()

	activator, err := activatorFor()
	if err != nil {
		return varNames, err
	}
	if err := activator.Unapprove(); err != nil {
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
	if removed {
		// Print unsets even when metadata cleanup fails after .envrc was removed.
		for _, name := range varNames {
			fmt.Printf("unset %s\n", name)
		}
	}
	if err != nil {
		return removed, varNames, err
	}
	if !removed {
		return false, nil, nil
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
	content := []byte(strings.Join(varNames, "\n") + "\n")
	if current, err := os.ReadFile(bwenvVarsCacheFile); err == nil && bytes.Equal(current, content) {
		return
	}
	_ = os.WriteFile(bwenvVarsCacheFile, content, 0600)
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
func Export(providerSlug string, folderName string, itemIDs []string) error {
	return ExportWithFolderID(providerSlug, folderName, "", itemIDs)
}

// ExportWithFolderID is the non-interactive export path used by generated
// .envrc files. An empty folderID keeps compatibility with older projects.
func ExportWithFolderID(providerSlug string, folderName string, folderID string, itemIDs []string) error {
	_, err := exportSecrets(providerSlug, folderName, folderID, itemIDs, false)
	return err
}

// ExportInteractive is like Export but allows interactive authentication
// (prompting for a master password). This is used by "bwenv allow" where
// the user explicitly runs bwenv and expects to enter their password once.
// Returns the session token so the caller can propagate it.
func ExportInteractive(providerSlug string, folderName string, itemIDs []string) (string, error) {
	return exportSecrets(providerSlug, folderName, "", itemIDs, true)
}

// exportSecrets is the shared implementation for Export and ExportInteractive.
// When interactive=false (direnv context), authentication failures produce a
// helpful error instead of blocking on a password prompt.
func exportSecrets(providerSlug string, folderName string, folderID string, itemIDs []string, interactive bool) (string, error) {
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
		// Non-interactive mode never probes, prompts, or syncs. The requested
		// provider operation is the session validation.
		session, err = p.AuthenticateNonInteractive()
	}
	if err != nil {
		printExportError("Authentication failed", err)
		return "", fmt.Errorf("authentication failed for %s: %w", p.Name(), err)
	}

	var targetFolder provider.Folder
	if folderID != "" {
		targetFolder = provider.Folder{ID: folderID, Name: folderName}
	} else {
		// Legacy projects resolve the folder by name for compatibility.
		folders, listErr := p.ListFolders(session)
		if listErr != nil {
			printExportError("Could not list folders", listErr)
			return session, fmt.Errorf("failed to list folders from %s: %w", p.Name(), listErr)
		}
		for _, f := range folders {
			if f.Name == folderName {
				targetFolder = f
				break
			}
		}
		if targetFolder.ID == "" {
			err := fmt.Errorf("folder %q not found (%d folders available)", folderName, len(folders))
			printExportError("Folder not found", err)
			return session, err
		}
	}

	// Fetch secrets — either all items in folder or specific items only.
	var secrets []provider.Secret
	if len(itemIDs) > 0 {
		secrets, err = p.GetSecretsByItemIDs(session, targetFolder, itemIDs)
	} else {
		secrets, err = p.GetSecrets(session, targetFolder)
	}
	if err != nil {
		printExportError("Could not fetch secrets", err)
		return session, fmt.Errorf("failed to get secrets from folder %q: %w", folderName, err)
	}

	// Collect variable names for the summary (before printing export lines).
	varNames := make([]string, 0, len(secrets))

	// Print each secret as an export statement to stdout.
	// direnv will eval this output to set the environment variables.
	for _, s := range secrets {
		key := shell.SanitizeKey(s.Key)
		fmt.Printf("export %s=%s\n", key, shell.Quote(s.Value))
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
		names = append(names, shell.SanitizeKey(s.Key))
	}
	return names, nil
}

// PreviewSecretsByIDs fetches secrets only from specific items and returns
// just the key names (not values). Used when the user selected individual
// items during "bwenv init" instead of loading the entire folder.
func PreviewSecretsByIDs(p provider.Provider, session string, folder provider.Folder, itemIDs []string) ([]string, error) {
	secrets, err := p.GetSecretsByItemIDs(session, folder, itemIDs)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(secrets))
	for _, s := range secrets {
		names = append(names, shell.SanitizeKey(s.Key))
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

	// Revoke the activation backend's approval. Non-fatal: the backend tooling
	// may not be installed when just cleaning up.
	if activator, aerr := activatorFor(); aerr == nil {
		_ = activator.Unapprove()
	}

	if err := os.Remove(".envrc"); err != nil {
		return false, varNames, fmt.Errorf("failed to remove .envrc: %w", err)
	}
	if err := os.Remove(".bwenv.toml"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return true, varNames, fmt.Errorf("failed to remove .bwenv.toml: %w", err)
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
//	│  ✅ 3 variable(s) loaded                 │
//	│    🔑 DB_USERNAME                         │
//	│    🔑 DB_PASSWORD                         │
//	│    🔑 API_TOKEN                           │
//	╰──────────────────────────────────────────╯
func printExportSummary(providerName string, folderName string, varNames []string) {
	var lines []string

	contextLine := summaryContext.Render(fmt.Sprintf("%s / %s", providerName, folderName))
	lines = append(lines, contextLine)

	// Empty separator line.
	lines = append(lines, "")

	if len(varNames) == 0 {
		// No variables found — show a warning.
		warningLine := summaryError.Render(emojiStr("⚠️", "[!]") + " No variables found in this folder")
		lines = append(lines, warningLine)
	} else {
		// Success line with count.
		countLine := summarySuccess.Render(fmt.Sprintf("%s %d variable(s) loaded", emojiStr("✅", "[OK]"), len(varNames)))
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

	errorLabel := summaryError.Render(emojiStr("❌", "[X]") + " " + label)
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
