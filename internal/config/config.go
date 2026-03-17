// Package config manages persistent user preferences for bwenv.
// Settings are stored in a JSON file at ~/.config/bwenv/config.json
// and control UI behavior like emoji display and direnv output visibility.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config holds all user-configurable preferences for bwenv.
// These settings are persisted to disk and loaded on every invocation.
type Config struct {
	// ShowEmoji controls whether emoji characters are displayed in output.
	// When false, emoji are replaced with plain-text equivalents.
	// Default: true
	ShowEmoji bool `json:"show_emoji"`

	// ShowDirenvOutput controls whether direnv loading/unloading messages
	// are visible to the user. When false (default), bwenv silences direnv
	// messages globally via DIRENV_LOG_FORMAT="".
	// Default: false
	ShowDirenvOutput bool `json:"show_direnv_output"`

	// ShowExportSummary controls whether the boxed summary is printed to
	// stderr every time direnv loads the .envrc (on every cd into the project).
	// Default: true
	ShowExportSummary bool `json:"show_export_summary"`

	// AutoSync controls whether bwenv runs "bw sync" before fetching secrets.
	// Disabling this can speed up loads if you sync manually.
	// Default: true
	AutoSync bool `json:"auto_sync"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ShowEmoji:         true,
		ShowDirenvOutput:  false,
		ShowExportSummary: true,
		AutoSync:          true,
	}
}

// configDir is the directory where the config file is stored.
// Follows XDG conventions: ~/.config/bwenv/
func configDir() (string, error) {
	// Prefer XDG_CONFIG_HOME if set.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bwenv"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", "bwenv"), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// ConfigPath returns the full path to the config file (exported for display).
func ConfigPath() (string, error) {
	return configPath()
}

// mu protects the cached config for concurrent access.
var (
	mu     sync.Mutex
	cached *Config
)

// Load reads the config from disk. If the file doesn't exist, it returns
// the default config without error. The result is cached for subsequent calls.
func Load() (Config, error) {
	mu.Lock()
	defer mu.Unlock()

	if cached != nil {
		return *cached, nil
	}

	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg, nil // Fallback to defaults silently.
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file yet — use defaults.
			cached = &cfg
			return cfg, nil
		}
		return cfg, fmt.Errorf("could not read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("could not parse config: %w", err)
	}

	cached = &cfg
	return cfg, nil
}

// Save writes the config to disk, creating the config directory if needed.
// It also invalidates the cache so the next Load() reads from disk.
func Save(cfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// Add a trailing newline for cleanliness.
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}

	// Invalidate cache so next Load() picks up the new values.
	cached = &cfg

	return nil
}

// Reset deletes the config file and resets to defaults.
func Reset() error {
	mu.Lock()
	defer mu.Unlock()

	cached = nil

	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove config file: %w", err)
	}

	return nil
}

// Invalidate clears the cached config so the next Load() reads from disk.
// Use this after external modifications to the config file.
func Invalidate() {
	mu.Lock()
	defer mu.Unlock()
	cached = nil
}

// Emoji returns the emoji string if ShowEmoji is true in the current config,
// otherwise returns the provided fallback string. This is a convenience helper
// so callers don't need to load the config themselves for every emoji.
func Emoji(emoji string, fallback string) string {
	cfg, _ := Load()
	if cfg.ShowEmoji {
		return emoji
	}
	return fallback
}
