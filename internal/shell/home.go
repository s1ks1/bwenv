package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ShortenHomePath replaces the user's home directory prefix with "~" for
// compact display. The returned path always uses forward slashes.
func ShortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home) {
		return filepath.ToSlash("~" + path[len(home):])
	}
	return path
}

// DetectRC returns the path to the user's primary shell RC file, derived from
// the SHELL environment variable.
func DetectRC() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	shellName := filepath.Base(os.Getenv("SHELL"))
	switch shellName {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		// On macOS, .bash_profile is preferred over .bashrc for login shells.
		if runtime.GOOS == "darwin" {
			profile := filepath.Join(home, ".bash_profile")
			if _, err := os.Stat(profile); err == nil {
				return profile, nil
			}
		}
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		// Fallback: zshrc (default on macOS), then bashrc.
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".zshrc"), nil
		}
		return filepath.Join(home, ".bashrc"), nil
	}
}
