package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/project"
	runshell "github.com/s1ks1/bwenv/v3/internal/shell"
)

// Activator is the experimental native-shell activation backend. It needs no
// external tool: the shell loads secrets by evaluating bwenv's output.
type Activator struct{}

func init() { activation.Register(&Activator{}) }

// Name is the activation.mode value that selects this backend.
func (a *Activator) Name() string { return "shell" }

// Available is always true: the backend is pure shell integration.
func (a *Activator) Available() bool { return true }

// EmitsExports reports that activation prints export statements for the shell.
func (a *Activator) EmitsExports() bool { return true }

// Detect reports whether the shell hook is installed in the user's RC file.
func (a *Activator) Detect() activation.Status {
	rc, err := runshell.DetectRC()
	if err != nil {
		return activation.Status{Installed: true, Detail: err.Error()}
	}
	display := runshell.ShortenHomePath(rc)

	content, err := os.ReadFile(rc)
	if err != nil {
		return activation.Status{Installed: true, Detail: "shell hook not installed in " + display}
	}
	if strings.Contains(string(content), hookMarker) {
		return activation.Status{Installed: true, Configured: true, Detail: "shell hook installed in " + display}
	}
	return activation.Status{Installed: true, Detail: "shell hook not installed in " + display}
}

// Render returns the shell hook snippet for the current shell.
func (a *Activator) Render(activation.Config) ([]byte, error) {
	snippet, err := Hook(DetectShell(os.Getenv("SHELL")))
	if err != nil {
		return nil, err
	}
	return []byte(snippet), nil
}

// Install appends the hook to the user's shell RC file. It is idempotent.
func (a *Activator) Install(activation.Config) error {
	snippet, err := Hook(DetectShell(os.Getenv("SHELL")))
	if err != nil {
		return err
	}
	rc, err := runshell.DetectRC()
	if err != nil {
		return err
	}
	display := runshell.ShortenHomePath(rc)

	content, err := os.ReadFile(rc)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not read %s: %w", display, err)
	}
	if strings.Contains(string(content), hookMarker) {
		return nil
	}

	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not write to %s: %w", display, err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + snippet + "\n"); err != nil {
		return fmt.Errorf("failed to append the shell hook to %s: %w", display, err)
	}
	return nil
}

// Remove is a no-op: editing arbitrary RC files to delete lines automatically is
// unsafe. Users remove the hook manually.
func (a *Activator) Remove() error { return nil }

// Resolve reads the canonical project config for the provider source.
func (a *Activator) Resolve() (activation.Source, error) {
	cfg, err := project.Load(".bwenv.toml")
	if err != nil {
		return activation.Source{}, err
	}
	return activation.Source{
		ProviderSlug: cfg.Provider,
		FolderName:   cfg.Project.FolderName,
		FolderID:     cfg.Project.FolderID,
		ItemIDs:      cfg.Project.Items,
	}, nil
}

// Approve, Unapprove and Reload are no-ops for the shell backend: there is no
// external tool to control, and the hook reacts to prompt events.
func (a *Activator) Approve() error   { return nil }
func (a *Activator) Unapprove() error { return nil }
func (a *Activator) Reload() error    { return nil }
