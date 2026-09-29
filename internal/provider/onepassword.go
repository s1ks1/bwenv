// Package provider — 1Password implementation.
// This file wraps the 1Password CLI ("op") to authenticate, list vaults,
// and retrieve secrets (fields) from vault items.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/s1ks1/bwenv/v3/internal/process"
)

// OnePassword implements the Provider interface using the 1Password CLI.
type OnePassword struct {
	Runner process.Runner

	// Warnings receives non-fatal per-item messages. A nil writer falls back
	// to os.Stderr; io.Discard silences them.
	Warnings io.Writer
}

func (o *OnePassword) withRunner(runner process.Runner) Provider {
	return &OnePassword{Runner: runner, Warnings: io.Discard}
}

// warnf writes a non-fatal message to the configured warnings writer.
func (o *OnePassword) warnf(format string, args ...any) {
	w := o.Warnings
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "warning: "+format+"\n", args...)
}

// reportProblems surfaces per-item failures without failing a whole fetch.
func (o *OnePassword) reportProblems(problems []string) {
	for _, problem := range problems {
		o.warnf("%s", problem)
	}
}

func (o *OnePassword) run(args []string, streams process.IO) (process.Result, error) {
	runner := o.Runner
	if runner == nil {
		runner = process.ExecRunner{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return runner.Run(ctx, "op", args, streams)
}

func (o *OnePassword) runInteractive(args []string, streams process.IO) (process.Result, error) {
	runner := o.Runner
	if runner == nil {
		runner = process.ExecRunner{}
	}
	return runner.Run(context.Background(), "op", args, streams)
}

// init registers the 1Password provider in the global registry on startup.
func init() {
	Register(&OnePassword{})
}

// Name returns the human-readable provider name.
func (o *OnePassword) Name() string { return "1Password" }

// Slug returns the short identifier used in CLI flags and .envrc files.
func (o *OnePassword) Slug() string { return "1password" }

// Description returns a brief explanation of this provider.
func (o *OnePassword) Description() string {
	return "Sync secrets from 1Password vaults (uses 'op' CLI)"
}

// CLICommand returns the CLI binary name that must be installed.
func (o *OnePassword) CLICommand() string { return "op" }

// IsAvailable checks whether the "op" CLI is installed and in PATH.
func (o *OnePassword) IsAvailable() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

// IsAuthenticated checks if the user has an active 1Password CLI session.
// The "op" CLI v2+ uses system authentication (biometrics, etc.) so we
// test by running a simple command and seeing if it succeeds.
func (o *OnePassword) IsAuthenticated() bool {
	result, err := o.run([]string{"vault", "list", "--format=json"}, process.IO{})
	if err != nil {
		return false
	}
	return len(result.Stdout) > 0
}

// Authenticate signs in to 1Password. With op CLI v2+, this typically
// triggers biometric or system authentication. For older versions or
// service accounts, the OP_SESSION_* or OP_SERVICE_ACCOUNT_TOKEN env
// vars may already be set. Returns an empty session string since op v2
// manages sessions internally.
func (o *OnePassword) Authenticate() (string, error) {
	// Check if already authenticated (op v2 uses system auth).
	if o.IsAuthenticated() {
		return "", nil
	}

	// Check for service account token (headless / CI environments).
	if token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"); token != "" {
		// Verify the token works.
		if _, err := o.run([]string{"vault", "list", "--format=json"}, process.IO{}); err == nil {
			return "", nil
		}
		return "", fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN is set but invalid")
	}

	// Attempt interactive sign-in. The op CLI v2 will open a system
	// authentication prompt (Touch ID, password dialog, etc.).
	_, err := o.runInteractive([]string{"signin"}, process.IO{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr})
	if err != nil {
		return "", fmt.Errorf("failed to sign in to 1Password: %w\n\nMake sure you have 'op' CLI v2+ installed and configured.\nSee: https://developer.1password.com/docs/cli/get-started/", err)
	}

	return "", nil
}

// AuthenticateNonInteractive leaves validation to the requested op command.
func (o *OnePassword) AuthenticateNonInteractive() (string, error) {
	return "", nil
}

// opVault is the JSON shape returned by "op vault list".
type opVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (v opVault) ToFolder() Folder {
	return Folder(v)
}

// ListFolders returns all vaults in the 1Password account.
// In 1Password, "vaults" are the equivalent of Bitwarden's "folders".
// The session parameter is unused for op v2 (auth is managed internally).
func (o *OnePassword) ListFolders(session string) ([]Folder, error) {
	result, err := o.run([]string{"vault", "list", "--format=json"}, process.IO{})
	if err != nil {
		return nil, fmt.Errorf("failed to list 1Password vaults: %w", err)
	}

	var vaults []opVault
	if err := json.Unmarshal(result.Stdout, &vaults); err != nil {
		return nil, fmt.Errorf("failed to parse vault list: %w", err)
	}

	folders := make([]Folder, 0, len(vaults))
	for _, v := range vaults {
		folders = append(folders, v.ToFolder())
	}

	return folders, nil
}

// opItem is the JSON shape for a 1Password item from "op item list".
type opItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// opItemDetail is the full JSON shape from "op item get" with all fields.
type opItemDetail struct {
	ID     string        `json:"id"`
	Title  string        `json:"title"`
	Fields []opItemField `json:"fields"`
}

// opItemField represents a single field on a 1Password item.
type opItemField struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	Type    string `json:"type"`    // e.g. "STRING", "CONCEALED", "OTP"
	Purpose string `json:"purpose"` // e.g. "USERNAME", "PASSWORD", "NOTES", or empty
}

// GetSecrets retrieves all fields from items in the given vault and returns
// them as key-value Secret pairs. Fields without a label are skipped.
// Built-in fields with purpose "NOTES" or system-generated fields (like OTP)
// are skipped unless they have a meaningful label. We focus on user-defined
// fields (sections) and the standard username/password fields.
//
// Items are fetched concurrently (up to 5 at a time) to minimize latency
// for vaults with many items.
func (o *OnePassword) GetSecrets(session string, folder Folder) ([]Secret, error) {
	items, err := o.listItemsInVault(folder.ID)
	if err != nil {
		return nil, err
	}

	itemIDs := make([]string, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	secrets, problems := o.fetchItemsSecrets(itemIDs, folder.ID)
	o.reportProblems(problems)
	return secrets, nil
}

// ListItems returns all items in the given vault.
func (o *OnePassword) ListItems(session string, folder Folder) ([]SecretItem, error) {
	items, err := o.listItemsInVault(folder.ID)
	if err != nil {
		return nil, err
	}

	secretItems := make([]SecretItem, 0, len(items))
	for _, item := range items {
		secretItems = append(secretItems, SecretItem{
			ID:   item.ID,
			Name: item.Title,
		})
	}

	return secretItems, nil
}

// GetSecretsByItemIDs retrieves fields only from the specified items.
// Item IDs are globally unique in 1Password, so no vault specification is
// needed. Unlike the best-effort folder fetch, a selected item that cannot be
// read is reported as an error so the caller never silently loads fewer secrets.
func (o *OnePassword) GetSecretsByItemIDs(session string, folder Folder, itemIDs []string) ([]Secret, error) {
	secrets, problems := o.fetchItemsSecrets(itemIDs, "")
	o.reportProblems(problems)
	if len(problems) > 0 {
		return secrets, fmt.Errorf("%d of %d selected item(s) could not be fetched", len(problems), len(itemIDs))
	}
	return secrets, nil
}

// listItemsInVault runs "op item list --vault" and returns the parsed items.
func (o *OnePassword) listItemsInVault(vaultID string) ([]opItem, error) {
	result, err := o.run([]string{"item", "list", "--vault", vaultID, "--format=json"}, process.IO{})
	if err != nil {
		return nil, fmt.Errorf("failed to list items in vault %q: %w", vaultID, err)
	}

	var items []opItem
	if err := json.Unmarshal(result.Stdout, &items); err != nil {
		return nil, fmt.Errorf("failed to parse item list: %w", err)
	}

	return items, nil
}

// fetchItemsSecrets fetches the given items concurrently (up to five at a time)
// and returns their secrets in the order of itemIDs, plus one message per item
// that could not be read. It is the single item-fetch path shared by the folder
// and selected-item operations. An empty vaultID omits the --vault flag.
func (o *OnePassword) fetchItemsSecrets(itemIDs []string, vaultID string) ([]Secret, []string) {
	const maxConcurrency = 5

	sem := make(chan struct{}, maxConcurrency)
	perItem := make([][]Secret, len(itemIDs))
	problems := make([]string, len(itemIDs))
	var wg sync.WaitGroup

	for i, id := range itemIDs {
		wg.Add(1)
		go func(idx int, itemID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			args := []string{"item", "get", itemID}
			if vaultID != "" {
				args = append(args, "--vault", vaultID)
			}
			args = append(args, "--format=json")

			result, err := o.run(args, process.IO{})
			if err != nil {
				problems[idx] = fmt.Sprintf("could not fetch item %q: %v", itemID, err)
				return
			}

			var detail opItemDetail
			if err := json.Unmarshal(result.Stdout, &detail); err != nil {
				problems[idx] = fmt.Sprintf("could not parse item %q: %v", itemID, err)
				return
			}
			perItem[idx] = fieldsToSecrets(detail.Fields)
		}(i, id)
	}

	wg.Wait()

	var secrets []Secret
	var issues []string
	for i := range itemIDs {
		secrets = append(secrets, perItem[i]...)
		if problems[i] != "" {
			issues = append(issues, problems[i])
		}
	}
	return secrets, issues
}

// fieldsToSecrets maps user-facing item fields to environment variables,
// skipping unlabeled fields, notes, one-time passwords and empty values.
func fieldsToSecrets(fields []opItemField) []Secret {
	var secrets []Secret
	for _, field := range fields {
		if field.Label == "" {
			continue
		}
		if strings.EqualFold(field.Purpose, "NOTES") {
			continue
		}
		if strings.EqualFold(field.Type, "OTP") {
			continue
		}
		if field.Value == "" {
			continue
		}
		secrets = append(secrets, Secret{Key: field.Label, Value: field.Value})
	}
	return secrets
}

// Lock signs out of the 1Password CLI session.
// For op CLI v2+, this runs "op signout" to terminate the current session.
// Returns nil if the sign-out succeeds or if there is no active session.
func (o *OnePassword) Lock() error {
	// If not authenticated, there's nothing to sign out of.
	if !o.IsAuthenticated() {
		return nil
	}

	_, err := o.run([]string{"signout"}, process.IO{Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		return fmt.Errorf("failed to sign out of 1Password: %w", err)
	}
	return nil
}
