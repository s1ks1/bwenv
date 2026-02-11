// Package provider — Bitwarden implementation.
// This file wraps the Bitwarden CLI ("bw") to authenticate, list folders,
// and retrieve secrets (custom fields) from vault items.
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Bitwarden implements the Provider interface using the Bitwarden CLI.
type Bitwarden struct{}

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
	cmd := exec.Command("bw", "list", "folders", "--session", session)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(out) > 0
}

// Authenticate unlocks the Bitwarden vault and returns a session token.
// It first runs "bw sync" to ensure the local cache is up to date,
// then prompts the user for their master password via "bw unlock".
func (b *Bitwarden) Authenticate() (string, error) {
	// Sync the vault first (best-effort, don't fail if offline).
	_ = runSilent("bw", "sync")

	// Check if there's already a valid session in the environment.
	if session := os.Getenv("BW_SESSION"); session != "" {
		cmd := exec.Command("bw", "list", "folders", "--session", session)
		if err := cmd.Run(); err == nil {
			return session, nil
		}
		// Session expired — fall through to unlock.
	}

	// Unlock the vault interactively. The "bw unlock --raw" command
	// prompts for the master password and outputs just the session token.
	cmd := exec.Command("bw", "unlock", "--raw")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to unlock Bitwarden vault: %w", err)
	}

	session := strings.TrimSpace(string(out))
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
func (b *Bitwarden) ListFolders(session string) ([]Folder, error) {
	cmd := exec.Command("bw", "list", "folders", "--session", session)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Bitwarden folders: %w", err)
	}

	var raw []bwFolder
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse folder list: %w", err)
	}

	folders := make([]Folder, 0, len(raw))
	for _, f := range raw {
		// Skip the "No Folder" entry (null name or empty).
		if f.Name == "" {
			continue
		}
		folders = append(folders, Folder{
			ID:   f.ID,
			Name: f.Name,
		})
	}

	return folders, nil
}

// bwItem is the JSON shape for a Bitwarden vault item.
type bwItem struct {
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
	cmd := exec.Command("bw", "list", "items", "--folderid", folder.ID, "--session", session)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list items in folder %q: %w", folder.Name, err)
	}

	var items []bwItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w", err)
	}

	var secrets []Secret
	for _, item := range items {
		for _, field := range item.Fields {
			// Only include fields that have both a name and a value.
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

// runSilent executes a command discarding all output.
// Returns any error from the command execution.
func runSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
