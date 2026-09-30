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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/config"
	"github.com/s1ks1/bwenv/v3/internal/output"
	"github.com/s1ks1/bwenv/v3/internal/project"
	"github.com/s1ks1/bwenv/v3/internal/provider"
	"github.com/s1ks1/bwenv/v3/internal/shell"
)

// Activate prepares the nearest project's activation artifact and reports the
// backend used. It works from any nested subdirectory and is idempotent:
// activating an already-active project succeeds without duplicating state.
func Activate() (backend string, err error) { return ActivateShell("bash") }

func ActivateShell(shellName string) (backend string, err error) {
	if err := validateShell(shellName); err != nil {
		return "", err
	}
	ctx := context.Background()

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

	activator, err := activation.ForProject()
	if err != nil {
		return "", err
	}

	// Backends that activate by emitting exports load the secrets now.
	if emitter, ok := activator.(activation.Emitter); ok && emitter.EmitsExports() {
		source, err := activator.Resolve()
		if err != nil {
			return "", err
		}
		if _, err := exportSecretsForShell(ctx, source.ProviderSlug, source.FolderName, source.FolderID, source.ItemIDs, false, shellName, true, false); err != nil {
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
func Deactivate() ([]string, error) { return DeactivateShell("bash") }

func DeactivateShell(shellName string) ([]string, error) {
	if err := validateShell(shellName); err != nil {
		return nil, err
	}
	if state, ok := os.LookupEnv(stateVariable); ok {
		return restoreState(state, shellName)
	}
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

// AllowAndExport is the handler for `eval "$(bwenv allow)"`. It:
//  1. Resolves the canonical project config (or a legacy direnv project).
//  2. Authenticates (may prompt for password ONCE).
//  3. Fetches secrets and prints export lines to stdout.
//  4. Exports BW_SESSION to the current shell, never to a project file.
//  5. Approves the selected activation backend.
func AllowAndExport() (providerSlug string, folderName string, err error) {
	return authAndExport()
}

func authAndExport() (providerSlug string, folderName string, err error) {
	ctx := context.Background()

	// Step 1: Resolve the project's activation backend and its secret source.
	activator, err := activation.ForProject()
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
	session, exportErr := exportSecretsForShell(ctx, providerSlug, folderName, source.FolderID, source.ItemIDs, true, currentShell(), activator.Name() == "shell", false)
	if exportErr != nil {
		return providerSlug, folderName, fmt.Errorf("export failed: %w", exportErr)
	}

	// Step 3: Output the fresh session token so the parent shell can keep it
	// for this session. It is never written to project files.
	if session != "" {
		fmt.Print(assignment("BW_SESSION", session, currentShell()))
	}

	userCfg, _ := config.Load()
	if activator.Name() == "direnv" && !userCfg.ShowDirenvOutput {
		fmt.Print(assignment("DIRENV_LOG_FORMAT", "", currentShell()))
		fmt.Print(assignment("DIRENV_WARN_TIMEOUT", "10m", currentShell()))
	}

	// Step 4: Approve the activation artifact after the session is available.
	if approveErr := activator.Approve(); approveErr != nil {
		output.Error(activator.Name()+" approval failed", approveErr)
	}

	return providerSlug, folderName, nil
}

// LoginAndExport is the handler for `eval "$(bwenv login)"`. It:
//  1. Resolves the configured project source.
//  2. Authenticates interactively (may prompt for master password).
//  3. Fetches secrets and prints export lines to stdout.
//  4. Exports the fresh BW_SESSION only to the current shell.
//  5. Approves the configured activation backend.
//
// This is functionally identical to AllowAndExport but semantically different:
// it's the recovery path when a session expires, while AllowAndExport is the
// initial approval path. Having a distinct "login" command makes the UX clearer.
func LoginAndExport() (providerSlug string, folderName string, err error) {
	return authAndExport()
}

// Refresh syncs providers that support it and asks the activation backend to
// reload the current project's environment. Secret values are not written.
func Refresh() (providerName string, synced bool, err error) {
	ctx := context.Background()
	activator, err := activation.ForProject()
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
	auth, authErr := provider.AsAuthenticator(p)
	if authErr != nil {
		return "", false, authErr
	}
	if !auth.IsAuthenticated(ctx) {
		return "", false, fmt.Errorf("%s session is not active; run 'bwenv login'", p.Name())
	}
	if syncer, ok := p.(provider.Syncer); ok {
		if err := syncer.Sync(ctx); err != nil {
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
	if state, ok := os.LookupEnv(stateVariable); ok {
		return restoreState(state, currentShell())
	}
	varNames := loadCachedVarNames()

	activator, err := activation.ForProject()
	if err != nil {
		return varNames, err
	}
	if err := activator.Unapprove(); err != nil {
		return varNames, err
	}

	// Print unset statements to stdout (captured by shell wrapper's eval).
	for _, name := range varNames {
		fmt.Print(unassignment(name, currentShell()))
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
		if state, ok := os.LookupEnv(stateVariable); ok {
			names, restoreErr := restoreState(state, currentShell())
			if err != nil {
				return removed, names, err
			}
			return removed, names, restoreErr
		}
		// Print unsets even when metadata cleanup fails after .envrc was removed.
		for _, name := range varNames {
			fmt.Print(unassignment(name, currentShell()))
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
			if validName(line) {
				names = append(names, line)
			}
		}
		if len(names) > 0 {
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
				if validName(key) && !seen[key] && !skip[key] {
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
// "export KEY=VALUE" lines to stdout. It also prints a concise summary
// to stderr showing which variables were loaded.
//
// This function is called by the activation backend. It NEVER prompts for a
// password interactively; the session must already be available in the current
// shell's BW_SESSION environment variable.
//
// stdout: only "export KEY=VALUE" lines (consumed by eval)
// stderr: concise status for the user (visible in the terminal)
func Export(ctx context.Context, providerSlug string, folderName string, itemIDs []string) error {
	return ExportWithFolderID(ctx, providerSlug, folderName, "", itemIDs)
}

// ExportWithFolderID is the non-interactive export path used by generated
// .envrc files. An empty folderID keeps compatibility with older projects.
func ExportWithFolderID(ctx context.Context, providerSlug string, folderName string, folderID string, itemIDs []string) error {
	_, err := exportSecrets(ctx, providerSlug, folderName, folderID, itemIDs, false)
	return err
}

// ExportQuiet is used by hooks that reevaluate on every prompt. Errors still
// return a failing exit status; explicit commands provide their diagnostics.
func ExportQuiet(ctx context.Context, providerSlug, folderName, folderID string, itemIDs []string) error {
	_, err := exportSecretsForShell(ctx, providerSlug, folderName, folderID, itemIDs, false, "bash", false, true)
	return err
}

// ExportInteractive is like Export but allows interactive authentication
// (prompting for a master password). This is used by "bwenv allow" where
// the user explicitly runs bwenv and expects to enter their password once.
// Returns the session token so the caller can propagate it.
func ExportInteractive(ctx context.Context, providerSlug string, folderName string, itemIDs []string) (string, error) {
	return exportSecrets(ctx, providerSlug, folderName, "", itemIDs, true)
}

// exportSecrets is the shared implementation for Export and ExportInteractive.
// When interactive=false (direnv context), authentication failures produce a
// helpful error instead of blocking on a password prompt.
func exportSecrets(ctx context.Context, providerSlug string, folderName string, folderID string, itemIDs []string, interactive bool) (string, error) {
	return exportSecretsForShell(ctx, providerSlug, folderName, folderID, itemIDs, interactive, "bash", false, false)
}

func exportSecretsForShell(ctx context.Context, providerSlug string, folderName string, folderID string, itemIDs []string, interactive bool, shellName string, remember bool, quiet bool) (string, error) {
	reportError := printExportError
	if quiet || interactive || remember {
		reportError = func(string, error) {}
	}
	// Load user preferences to decide whether to show the export summary.
	userCfg, _ := config.Load()

	// Look up the requested provider from the registry.
	p, err := provider.Get(providerSlug)
	if err != nil {
		reportError("Provider not found", err)
		return "", err
	}

	// Check that the provider's CLI tool is available on this system.
	if !p.IsAvailable() {
		err := fmt.Errorf("'%s' CLI is not installed", p.CLICommand())
		reportError(fmt.Sprintf("%s unavailable", p.Name()), err)
		return "", err
	}

	// Authenticate with the provider.
	auth, authErr := provider.AsAuthenticator(p)
	if authErr != nil {
		reportError("Provider unsupported", authErr)
		return "", authErr
	}
	var session string
	if interactive {
		// Interactive mode (bwenv allow): may prompt for master password.
		session, err = auth.Authenticate(ctx)
	} else {
		// Non-interactive mode never probes, prompts, or syncs. The requested
		// provider operation is the session validation.
		session, err = auth.AuthenticateNonInteractive(ctx)
	}
	if err != nil {
		reportError("Authentication failed", err)
		return "", fmt.Errorf("authentication failed for %s: %w", p.Name(), err)
	}

	var targetFolder provider.Folder
	if folderID != "" {
		targetFolder = provider.Folder{ID: folderID, Name: folderName}
	} else {
		// Legacy projects resolve the folder by name for compatibility.
		lister, listerErr := provider.AsFolderLister(p)
		if listerErr != nil {
			reportError("Provider unsupported", listerErr)
			return session, listerErr
		}
		folders, listErr := lister.ListFolders(ctx, session)
		if listErr != nil {
			reportError("Could not list folders", listErr)
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
			reportError("Folder not found", err)
			return session, err
		}
	}

	// Fetch secrets — either all items in folder or specific items only.
	fetcher, fetchErr := provider.AsSecretFetcher(p)
	if fetchErr != nil {
		reportError("Provider unsupported", fetchErr)
		return session, fetchErr
	}
	var secrets []provider.Secret
	if len(itemIDs) > 0 {
		secrets, err = fetcher.GetSecretsByItemIDs(ctx, session, targetFolder, itemIDs)
	} else {
		secrets, err = fetcher.GetSecrets(ctx, session, targetFolder)
	}
	if err != nil {
		reportError("Could not fetch secrets", err)
		return session, fmt.Errorf("failed to get secrets from folder %q: %w", folderName, err)
	}

	if remember {
		state, err := rememberState(secrets)
		if err != nil {
			return session, err
		}
		fmt.Print(assignment(stateVariable, state, shellName))
	}

	// A hook may reload immediately after login. Report only changed values.
	changed := false
	// Collect variable names for the summary (before printing export lines).
	varNames := make([]string, 0, len(secrets))

	// Print each secret as an export statement to stdout.
	// direnv will eval this output to set the environment variables.
	for _, s := range secrets {
		key := shell.SanitizeKey(s.Key)
		value, exists := os.LookupEnv(key)
		changed = changed || !exists || value != s.Value
		fmt.Print(assignment(key, s.Value, shellName))
		varNames = append(varNames, key)
	}

	// Cache variable names so disallow/remove can unset them later
	// without needing to re-authenticate with the provider.
	saveVarNamesCache(varNames)

	// Print a concise summary to stderr so the user sees what happened.
	// This goes to stderr to avoid polluting the eval'd stdout.
	// Controlled by the ShowExportSummary config preference.
	if !quiet && userCfg.ShowExportSummary && (interactive || changed) {
		printExportSummary(p.Name(), folderName, varNames)
	}

	return session, nil
}

// ── Secret preview ──────────────────────────────────────────────────────────

// PreviewSecrets fetches secrets from the given provider and folder and returns
// just the key names (not values). This is used during "bwenv init" to show
// the user what variables will be loaded, without exposing actual secret values.
func PreviewSecrets(ctx context.Context, p provider.SecretFetcher, session string, folder provider.Folder) ([]string, error) {
	secrets, err := p.GetSecrets(ctx, session, folder)
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
func PreviewSecretsByIDs(ctx context.Context, p provider.SecretFetcher, session string, folder provider.Folder, itemIDs []string) ([]string, error) {
	secrets, err := p.GetSecretsByItemIDs(ctx, session, folder, itemIDs)
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
	activator, err := activation.ForProject()
	if err != nil {
		if _, statErr := os.Stat(".envrc"); statErr != nil {
			return false, nil, err
		}
		activator, err = activation.Get("direnv")
		if err != nil {
			return false, nil, err
		}
	}
	if _, err := os.Stat(".bwenv.toml"); os.IsNotExist(err) {
		if _, err := os.Stat(".envrc"); os.IsNotExist(err) {
			return false, nil, nil
		}
	}
	varNames := loadCachedVarNames()
	if err := activator.Remove(); err != nil {
		return false, varNames, err
	}
	if err := os.Remove(".bwenv.toml"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return true, varNames, fmt.Errorf("failed to remove .bwenv.toml: %w", err)
	}

	// Also clean up the variable name cache.
	removeCachedVarNames()

	return true, varNames, nil
}

// printExportSummary shows the result without secret values or repeated branding.
func printExportSummary(providerName string, folderName string, varNames []string) {
	context := providerName + " / " + folderName
	if len(varNames) == 0 {
		output.Warning("No variables found · " + context)
		return
	}
	noun := "variables"
	if len(varNames) == 1 {
		noun = "variable"
	}
	output.Success(fmt.Sprintf("%d %s loaded · %s", len(varNames), noun, context))
}

func printExportError(label string, err error) {
	if errors.Is(err, provider.ErrSessionExpired) || errors.Is(err, provider.ErrNotAuthenticated) {
		output.Warning("Session locked or expired · run bwenv login")
		return
	}
	output.Error(label, err)
}
