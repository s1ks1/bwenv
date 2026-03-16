// Package provider — Bitwarden implementation.
// This file wraps the Bitwarden CLI ("bw") to authenticate, list folders,
// and retrieve secrets (custom fields) from vault items.
package provider

import (
	"bytes"
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
	// Capture both stdout and stderr so we can detect error responses.
	cmd := exec.Command("bw", "list", "folders", "--session", session)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false
	}
	out := bytes.TrimSpace(stdout.Bytes())
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
	_ = runSilent("bw", "sync")

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
// Sync is NOT called here — it's done once in Authenticate() to avoid
// redundant network calls.
func (b *Bitwarden) ListFolders(session string) ([]Folder, error) {
	cmd := exec.Command("bw", "list", "folders", "--session", session)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("failed to list Bitwarden folders: %s", stderrStr)
		}
		return nil, fmt.Errorf("failed to list Bitwarden folders: %w (is your session still valid? try 'bwenv login' to re-authenticate)", err)
	}

	out := bytes.TrimSpace(stdout.Bytes())

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
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("failed to list items in folder %q: %s", folder.Name, stderrStr)
		}
		return nil, fmt.Errorf("failed to list items in folder %q: %w", folder.Name, err)
	}

	out := bytes.TrimSpace(stdout.Bytes())

	// Guard against empty output.
	if len(out) == 0 {
		return nil, fmt.Errorf(
			"Bitwarden CLI returned empty output for folder %q.\n"+
				"    Your session may have expired. Run 'bwenv login' to re-authenticate",
			folder.Name)
	}

	// Verify the output starts with '['.
	if out[0] != '[' {
		preview := string(out)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf(
			"Bitwarden CLI returned unexpected output for folder %q (expected JSON array):\n    %s",
			folder.Name, preview)
	}

	var items []bwItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("failed to parse items: %w\n    Raw output: %s", err, truncateOutput(out))
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

// Lock locks the Bitwarden vault, invalidating the current session.
// This is used by the "bwenv logout" command.
func (b *Bitwarden) Lock() error {
	cmd := exec.Command("bw", "lock")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to lock Bitwarden vault: %w", err)
	}
	return nil
}

// runSilent executes a command discarding all output.
// Returns any error from the command execution.
func runSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
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
