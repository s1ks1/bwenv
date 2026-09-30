package project

import (
	"errors"
	"os"
	"path/filepath"
)

// FindRoot walks up from dir and returns the nearest directory that contains a
// .bwenv.toml or a legacy .envrc. It returns "" and no error when no project is
// found in dir or any of its parents.
//
// This is what makes activation work from a nested subdirectory: the caller
// operates on the project root, not on the current directory.
func FindRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		for _, marker := range []string{".bwenv.toml", ".envrc"} {
			if _, err := os.Stat(filepath.Join(abs, marker)); err == nil {
				return abs, nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", nil // reached the filesystem root
		}
		abs = parent
	}
}
