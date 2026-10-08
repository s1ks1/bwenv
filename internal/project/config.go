// Package project owns the canonical, secret-free .bwenv.toml project
// definition: loading, validation and encoding. Provider references and the
// activation mode live here; secret values never do.
package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ConfigVersion is the schema version written to and expected from .bwenv.toml.
const ConfigVersion = 1

// Config is the canonical project definition stored in .bwenv.toml.
type Config struct {
	Version    int        `toml:"version"`
	Provider   string     `toml:"provider"`
	Project    Metadata   `toml:"project"`
	Activation Activation `toml:"activation"`
}

// Metadata identifies which provider folder or vault a project reads from.
type Metadata struct {
	FolderID   string   `toml:"folder_id,omitempty"`
	FolderName string   `toml:"folder_name"`
	Items      []string `toml:"items,omitempty"`
}

// Activation selects how the project's environment is activated.
type Activation struct {
	Mode     string `toml:"mode"`
	Disabled bool   `toml:"disabled,omitempty"`
}

// SetDisabled persists explicit approval for native/mise projects. An absent
// disabled field keeps existing projects enabled; ordinary directory exits do
// not change this setting.
func SetDisabled(path string, disabled bool) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	if cfg.Activation.Disabled == disabled {
		return nil
	}
	cfg.Activation.Disabled = disabled
	return Write(path, cfg)
}

// Load reads and validates a canonical bwenv project file. A directory path
// (the documented `--project .` form) resolves to <dir>/.bwenv.toml.
func Load(path string) (Config, error) {
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		path = filepath.Join(path, ".bwenv.toml")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read project config %s: %w", path, err)
	}

	var cfg Config
	decoder := toml.NewDecoder(bytes.NewReader(content)).DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse project config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid project config %s: %w", path, err)
	}
	return cfg, nil
}

// Validate rejects configurations bwenv cannot act on. It is applied on both
// load and encode so a hand-edited file and a generated file obey the same rules.
func (cfg Config) Validate() error {
	if cfg.Version != ConfigVersion {
		return fmt.Errorf("unsupported version %d (supported: %d)", cfg.Version, ConfigVersion)
	}
	if strings.TrimSpace(cfg.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(cfg.Project.FolderName) == "" {
		return fmt.Errorf("project.folder_name is required")
	}
	if strings.TrimSpace(cfg.Activation.Mode) == "" {
		return fmt.Errorf("activation.mode is required")
	}
	for _, itemID := range cfg.Project.Items {
		if strings.TrimSpace(itemID) == "" {
			return fmt.Errorf("project.items must not contain empty IDs")
		}
	}
	return nil
}

// Encode validates and marshals a config to TOML.
func Encode(cfg Config) ([]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	content, err := toml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode project config: %w", err)
	}
	return content, nil
}

// Write encodes cfg and writes it to path.
func Write(path string, cfg Config) error {
	content, err := Encode(cfg)
	if err != nil {
		return fmt.Errorf("could not write project config: %w", err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("secure %s: %w", path, err)
	}
	return nil
}
