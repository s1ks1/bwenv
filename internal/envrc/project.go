package envrc

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/s1ks1/bwenv/internal/provider"
)

const projectConfigVersion = 1

// ProjectConfig contains provider references and activation settings, never secrets.
type ProjectConfig struct {
	Version    int              `toml:"version"`
	Provider   string           `toml:"provider"`
	Project    ProjectMetadata  `toml:"project"`
	Activation ActivationConfig `toml:"activation"`
}

type ProjectMetadata struct {
	FolderID   string   `toml:"folder_id,omitempty"`
	FolderName string   `toml:"folder_name"`
	Items      []string `toml:"items,omitempty"`
}

type ActivationConfig struct {
	Mode string `toml:"mode"`
}

// LoadProjectConfig reads and validates a canonical bwenv project file.
func LoadProjectConfig(path string) (ProjectConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("read project config %s: %w", path, err)
	}

	var cfg ProjectConfig
	decoder := toml.NewDecoder(bytes.NewReader(content)).DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("parse project config %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return ProjectConfig{}, fmt.Errorf("invalid project config %s: %w", path, err)
	}
	return cfg, nil
}

func (cfg ProjectConfig) validate() error {
	if cfg.Version != projectConfigVersion {
		return fmt.Errorf("unsupported version %d (supported: %d)", cfg.Version, projectConfigVersion)
	}
	if _, err := provider.Get(cfg.Provider); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Project.FolderName) == "" {
		return fmt.Errorf("project.folder_name is required")
	}
	if cfg.Activation.Mode != "direnv" {
		return fmt.Errorf("unsupported activation.mode %q (supported: direnv)", cfg.Activation.Mode)
	}
	for _, itemID := range cfg.Project.Items {
		if strings.TrimSpace(itemID) == "" {
			return fmt.Errorf("project.items must not contain empty IDs")
		}
	}
	return nil
}

func writeProjectConfig(cfg Config) error {
	projectConfig := ProjectConfig{
		Version:  projectConfigVersion,
		Provider: cfg.ProviderSlug,
		Project: ProjectMetadata{
			FolderID:   cfg.FolderID,
			FolderName: cfg.FolderName,
			Items:      cfg.ItemIDs,
		},
		Activation: ActivationConfig{Mode: "direnv"},
	}
	content, err := encodeProjectConfig(projectConfig)
	if err != nil {
		return fmt.Errorf("could not write project config: %w", err)
	}
	if err := os.WriteFile(".bwenv.toml", content, 0644); err != nil {
		return fmt.Errorf("write .bwenv.toml: %w", err)
	}
	return nil
}

func encodeProjectConfig(cfg ProjectConfig) ([]byte, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	content, err := toml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode project config: %w", err)
	}
	return content, nil
}
