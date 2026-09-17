// Package provider — Bitwarden implementation.
// This file wraps the Bitwarden CLI ("bw") to authenticate, list folders,
// and retrieve secrets (custom fields) from vault items.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/s1ks1/bwenv/internal/process"
)

// Bitwarden implements the Provider interface using the Bitwarden CLI.
type Bitwarden struct{ Runner process.Runner }

func (b *Bitwarden) withRunner(runner process.Runner) Provider {
	return &Bitwarden{Runner: runner}
}

func (b *Bitwarden) run(args []string, streams process.IO) (process.Result, error) {
	runner := b.Runner
	if runner == nil {
		runner = process.ExecRunner{}
	}
	return runner.Run(context.Background(), "bw", args, streams)
}

// init registers the Bitwarden provider in the global registry on startup.
func init() {
	Register(&Bitwarden{})
}

// Name returns the human-readable provider name.
func (b *Bitwarden) Name() string { return "Bitwarden" }

// Slug returns the short identifier used in CLI flags and .envrc files.
func (b *Bitwarden) Slug() string { return "bitwarden" }

// Description returns a brief explanation of this provider.
func (b *Bitwarden) Description() string {
	return "Sync secrets from Bitwarden vault folders (uses 'bw' CLI)"
}

// CLICommand returns the CLI binary name that must be installed.
func (b *Bitwarden) CLICommand() string { return "bw" }

// IsAvailable checks whether the "bw" CLI is installed and in PATH.
func (b *Bitwarden) IsAvailable() bool {
	_, err := exec.LookPath("bw")
	return err == nil
}

// IsAuthenticated checks if there is a valid BW_SESSION environment variable
// and if the session can actually reach the vault.
func (b *Bitwarden) IsAuthenticated() bool {
	session := os.Getenv("BW_SESSION")
	if session == "" {
		return false
	}
	// Try listing folders to verify the session is still valid.
	// Capture both stdout and stderr so we can detect error responses.
	result, err := b.run([]string{"list", "folders", "--session", session}, process.IO{})
	if err != nil {
		return false
	}
	out := bytes.TrimSpace(result.Stdout)
	// The output must be a non-empty JSON array to be considered valid.
	return len(out) > 0 && out[0] == '['
}

// Authenticate unlocks the Bitwarden vault and returns a session token.
// If BW_SESSION is already set and valid, it reuses it without prompting.
// Otherwise, it syncs the vault and prompts the user for their master password.
func (b *Bitwarden) Authenticate() (string, error) {
	// Check if there's already a valid session in the environment.
	if session := os.Getenv("BW_SESSION"); session != "" {
		if b.IsAuthenticated() {
			return session, nil
		}
		// Session expired — fall through to unlock.
	}

	// Sync the vault first (best-effort, don't fail if offline).
	_, _ = b.run([]string{"sync"}, process.IO{Stdout: io.Discard, Stderr: io.Discard})

	// Unlock the vault interactively. The "bw unlock --raw" command
	// prompts for the master password and outputs just the session token.
	result, err := b.run([]string{"unlock", "--raw"}, process.IO{Stdin: os.Stdin, Stderr: os.Stderr})
	if err != nil {
		return "", fmt.Errorf("failed to unlock Bitwarden vault: %w", err)
	}

	session := strings.TrimSpace(string(result.Stdout))
	if session == "" {
		return "", fmt.Errorf("received empty session token from 'bw unlock'")
	}

	return session, nil
}

// bwFolder is the JSON shape returned by "bw list folders".
type bwFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListFolders returns all folders in the Bitwarden vault.
// Sync is NOT called here — it's done once in Authenticate() to avoid
// redundant network calls.
func (b *Bitwarden) ListFolders(session string) ([]Folder, error) {
	result, err := b.run([]string{"list", "folders", "--session", session}, process.IO{})
	if err != nil {
		stderrStr := strings.TrimSpace(string(result.Stderr))
		if stderrStr != "" {
			return nil, fmt.Errorf("failed to list Bitwarden folders: %s", stderrStr)
		}
		return nil, fmt.Errorf("failed to list Bitwarden folders: %w (is your session still valid? try 'bwenv login' to re-authenticate)", err)
	}

	out := bytes.TrimSpace(result.Stdout)

	// Guard against empty output — this can happen when the session has
	// expired or the vault is locked. The bw CLI sometimes exits 0 but
	// produces no JSON output (or outputs an error message to stdout).
	if len(out) == 0 {
		return nil, fmt.Errorf(
			"Bitwarden CLI returned empty output when listing folders.\n" +
				"    This usually means your session has expired.\n" +
				"    Run 'bwenv login' to re-authenticate")
	}

	// Verify the output looks like a JSON array before parsing.
	// The bw CLI can sometimes return an error message as plain text
	// (e.g. "Your vault is locked.") instead of JSON.
	if out[0] != '[' {
		// Try to give a helpful message from whatever bw returned.
		preview := string(out)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf(
			"Bitwarden CLI returned unexpected output (expected JSON array):\n    %s\n"+
				"    Your session may have expired. Run 'bwenv login' to re-authenticate",
			preview)
	}

	var raw []bwFolder
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse folder list: %w\n    Raw output: %s", err, truncateOutput(out))
	}

	folders := make([]Folder, 0, len(raw))
	for _, f := range raw {
		// Skip the "No Folder" entry (null name or empty).
		if f.Name == "" {
			continue
		}
		folders = append(folders, Folder(f))
	}

	return folders, nil
}

// bwItem is the JSON shape for a Bitwarden vault item.
type bwItem struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Fields []bwField `json:"fields"`
}

// bwField is a custom field on a Bitwarden item.
type bwField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"` // 0 = text, 1 = hidden, 2 = boolean
}

// GetSecrets retrieves all custom fields from items in the given folder
// and returns them as key-value Secret pairs. Each field becomes one
// environment variable — the field name is the key, the field value is the value.
func (b *Bitwarden) GetSecrets(session string, folder Folder) ([]Secret, error) {
	items, err := b.listItems(session, folder.ID)
	if err != nil {
		return nil, err
	}

	var secrets []Secret
	for _, item := range items {
		for _, field := range item.Fields {
			if field.Name == "" {
				continue
			}
			secrets = append(secrets, Secret{
				Key:   field.Name,
				Value: field.Value,
			})
		}
	}

	return secrets, nil
}

// ListItems returns all items in the given folder. Each item is a
// single vault entry that may contain multiple custom fields.
func (b *Bitwarden) ListItems(session string, folder Folder) ([]SecretItem, error) {
	items, err := b.listItems(session, folder.ID)
	if err != nil {
		return nil, err
	}

	secretItems := make([]SecretItem, 0, len(items))
	for _, item := range items {
		secretItems = append(secretItems, SecretItem{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	return secretItems, nil
}

// GetSecretsByItemIDs retrieves custom fields only from the specified items.
func (b *Bitwarden) GetSecretsByItemIDs(session string, itemIDs []string) ([]Secret, error) {
	var secrets []Secret

	for _, id := range itemIDs {
		result, err := b.run([]string{"get", "item", id, "--session", session}, process.IO{})
		if err != nil {
			stderrStr := strings.TrimSpace(string(result.Stderr))
			if stderrStr != "" {
				return nil, fmt.Errorf("failed to get item %q: %s", id, stderrStr)
			}
			return nil, fmt.Errorf("failed to get item %q: %w", id, err)
		}

		out := bytes.TrimSpace(result.Stdout)
		if len(out) == 0 {
			continue
		}

		var item bwItem
		if err := json.Unmarshal(out, &item); err != nil {
			return nil, fmt.Errorf("failed to parse item %q: %w\n    Raw: %s", id, err, truncateOutput(out))
		}

		for _, field := range item.Fields {
			if field.Name == "" {
				continue
			}
			secrets = append(secrets, Secret{
				Key:   field.Name,
				Value: field.Value,
			})
		}
	}

	return secrets, nil
}

// listItems is the shared implementation that parses the raw bwItem list
// from "bw list items". Used by both GetSecrets and ListItems.
func (b *Bitwarden) listItems(session string, folderID string) ([]bwItem, error) {
	result, err := b.run([]string{"list", "items", "--folderid", folderID, "--session", session}, process.IO{})
	if err != nil {
		stderrStr := strings.TrimSpace(string(result.Stderr))
		if stderrStr != "" {
			return nil, fmt.Errorf("failed to list items in folder %q: %s", folderID, stderrStr)
		}
		return nil, fmt.Errorf("failed to list items in folder %q: %w", folderID, err)
	}

	out := bytes.TrimSpace(result.Stdout)

	if len(out) == 0 {
		return nil, fmt.Errorf(
			"Bitwarden CLI returned empty output for folder %q.\n"+
				"    Your session may have expired. Run 'bwenv login' to re-authenticate",
			folderID)
	}

	if out[0] != '[' {
		preview := string(out)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf(
			"Bitwarden CLI returned unexpected output for folder %q (expected JSON array):\n    %s",
			folderID, preview)
	}

	var items []bwItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w\n    Raw output: %s", err, truncateOutput(out))
	}

	return items, nil
}

// Lock locks the Bitwarden vault, invalidating the current session.
// This is used by the "bwenv logout" command.
func (b *Bitwarden) Lock() error {
	_, err := b.run([]string{"lock"}, process.IO{Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		return fmt.Errorf("failed to lock Bitwarden vault: %w", err)
	}
	return nil
}

// truncateOutput returns a truncated string representation of raw bytes
// for use in error messages. Limits output to 300 characters.
func truncateOutput(data []byte) string {
	s := string(data)
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}
