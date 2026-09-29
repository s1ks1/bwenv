package direnv

import (
	"os"
	"os/exec"

	"github.com/s1ks1/bwenv/v3/internal/activation"
)

// Activator is the direnv backend of the activation boundary. It is the stable
// default: it generates .envrc files and controls the direnv CLI.
type Activator struct{}

func init() { activation.Register(&Activator{}) }

// Name is the activation.mode value that selects this backend.
func (a *Activator) Name() string { return "direnv" }

// Available reports whether the direnv CLI is installed.
func (a *Activator) Available() bool {
	_, err := exec.LookPath("direnv")
	return err == nil
}

// Detect reports whether direnv is installed and the project has an .envrc.
func (a *Activator) Detect() activation.Status {
	status := activation.Status{Installed: a.Available()}
	if _, err := os.Stat(".envrc"); err == nil {
		status.Configured = true
	}
	switch {
	case !status.Installed:
		status.Detail = "direnv is not installed"
	case !status.Configured:
		status.Detail = "no .envrc in this directory"
	default:
		status.Detail = ".envrc present"
	}
	return status
}

// Render returns the .envrc content for cfg without writing anything.
func (a *Activator) Render(cfg activation.Config) ([]byte, error) { return Render(cfg) }

// Install writes .bwenv.toml (when a folder ID exists) and the .envrc.
func (a *Activator) Install(cfg activation.Config) error { return Install(cfg) }

// Remove deletes the .envrc and revokes direnv's approval.
func (a *Activator) Remove() error { return Remove() }

// Resolve reports the provider source recorded in .bwenv.toml or a legacy .envrc.
func (a *Activator) Resolve() (activation.Source, error) {
	providerSlug, folderName, folderID, itemIDs, err := ParseConfigWithFolderID()
	if err != nil {
		return activation.Source{}, err
	}
	return activation.Source{
		ProviderSlug: providerSlug,
		FolderName:   folderName,
		FolderID:     folderID,
		ItemIDs:      itemIDs,
	}, nil
}

// Approve runs "direnv allow" for the current project.
func (a *Activator) Approve() error { return Allow() }

// Unapprove runs "direnv deny" for the current project.
func (a *Activator) Unapprove() error { return Disallow() }

// Reload asks direnv to reload the current project's environment.
func (a *Activator) Reload() error { return Reload() }
