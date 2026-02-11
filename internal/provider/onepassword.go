// Package provider — 1Password implementation.
// This file wraps the 1Password CLI ("op") to authenticate, list vaults,
// and retrieve secrets (fields) from vault items.
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// OnePassword implements the Provider interface using the 1Password CLI.
type OnePassword struct{}

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
	cmd := exec.Command("op", "vault", "list", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(out) > 0
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
		cmd := exec.Command("op", "vault", "list", "--format=json")
		if err := cmd.Run(); err == nil {
			return "", nil
		}
		return "", fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN is set but invalid")
	}

	// Attempt interactive sign-in. The op CLI v2 will open a system
	// authentication prompt (Touch ID, password dialog, etc.).
	cmd := exec.Command("op", "signin")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to sign in to 1Password: %w\n\nMake sure you have 'op' CLI v2+ installed and configured.\nSee: https://developer.1password.com/docs/cli/get-started/", err)
	}

	return "", nil
}

// opVault is the JSON shape returned by "op vault list".
type opVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListFolders returns all vaults in the 1Password account.
// In 1Password, "vaults" are the equivalent of Bitwarden's "folders".
// The session parameter is unused for op v2 (auth is managed internally).
func (o *OnePassword) ListFolders(session string) ([]Folder, error) {
	cmd := exec.Command("op", "vault", "list", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list 1Password vaults: %w", err)
	}

	var vaults []opVault
	if err := json.Unmarshal(out, &vaults); err != nil {
		return nil, fmt.Errorf("failed to parse vault list: %w", err)
	}

	folders := make([]Folder, 0, len(vaults))
	for _, v := range vaults {
		folders = append(folders, Folder{
			ID:   v.ID,
			Name: v.Name,
		})
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
func (o *OnePassword) GetSecrets(session string, folder Folder) ([]Secret, error) {
	// Step 1: List all items in the vault.
	listCmd := exec.Command("op", "item", "list", "--vault", folder.ID, "--format=json")
	listOut, err := listCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list items in vault %q: %w", folder.Name, err)
	}

	var items []opItem
	if err := json.Unmarshal(listOut, &items); err != nil {
		return nil, fmt.Errorf("failed to parse item list: %w", err)
	}

	// Step 2: Fetch full details for each item and extract fields.
	var secrets []Secret
	for _, item := range items {
		getCmd := exec.Command("op", "item", "get", item.ID, "--vault", folder.ID, "--format=json")
		getOut, err := getCmd.Output()
		if err != nil {
			// Log a warning but continue with other items.
			fmt.Fprintf(os.Stderr, "warning: could not fetch item %q: %v\n", item.Title, err)
			continue
		}

		var detail opItemDetail
		if err := json.Unmarshal(getOut, &detail); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not parse item %q: %v\n", item.Title, err)
			continue
		}

		for _, field := range detail.Fields {
			// Skip fields without a label — they can't be mapped to env var names.
			if field.Label == "" {
				continue
			}

			// Skip the "notes" purpose field (typically large text, not a secret).
			if strings.EqualFold(field.Purpose, "NOTES") {
				continue
			}

			// Skip OTP fields — they are time-based and not useful as static env vars.
			if strings.EqualFold(field.Type, "OTP") {
				continue
			}

			// Skip fields with empty values.
			if field.Value == "" {
				continue
			}

			secrets = append(secrets, Secret{
				Key:   field.Label,
				Value: field.Value,
			})
		}
	}

	return secrets, nil
}
