package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/activation/direnv"
	_ "github.com/s1ks1/bwenv/v3/internal/activation/mise"
	"github.com/s1ks1/bwenv/v3/internal/activation/shell"
	"github.com/s1ks1/bwenv/v3/internal/benchmark"
	"github.com/s1ks1/bwenv/v3/internal/export"
	"github.com/s1ks1/bwenv/v3/internal/project"
	"github.com/s1ks1/bwenv/v3/internal/provider"
	_ "github.com/s1ks1/bwenv/v3/internal/provider/all"
	"github.com/s1ks1/bwenv/v3/internal/session"
	"github.com/s1ks1/bwenv/v3/internal/ui"
)

// Request contains parsed command options. Application workflows do not read os.Args.
type Request struct {
	Command, Version, Provider, Folder, FolderID, Project, Activation, Shell string
	Items                                                                    []string
	Quiet, ShellOnly, Fingerprint, DryRun                                    bool
}

// Run coordinates a command and returns its process exit status.
func Run(request Request) int {
	if request.Shell == "" || request.Shell == "auto" {
		request.Shell = shell.DetectShell(os.Getenv("SHELL"))
	}
	switch request.Command {
	case "benchmark":
		return runBenchmark(request)
	case "init":
		return runInit(request)
	case "export":
		return runExport(request)
	case "allow":
		return runAllow(request)
	case "disallow":
		return runDisallow(request)
	case "remove":
		return runRemove(request)
	case "config":
		return runConfig(request)
	case "logout":
		return runLogout(request)
	case "login":
		return runLogin(request)
	case "refresh":
		return runRefresh(request)
	case "activate":
		return runActivate(request)
	case "deactivate":
		return runDeactivate(request)
	case "root":
		return runRoot(request)
	case "hook":
		return runHook(request)
	case "status":
		return runStatus(request)
	case "doctor":
		return runDoctor(request)
	case "migrate":
		return runMigrate(request)
	case "login-hint":
		ui.PrintWarning("Variables are not loaded · run bwenv login")
	case "version":
		ui.PrintVersion(request.Version)
	case "examples":
		ui.PrintExamples(request.Version)
	case "help":
		ui.PrintUsage(request.Version)
	default:
		ui.PrintError("Unknown command", fmt.Errorf("%s", request.Command))
		return 1
	}
	return 0
}

func runBenchmark(request Request) int {
	providerSlug, folder, folderID, itemIDs := request.Provider, request.Folder, request.FolderID, request.Items
	if providerSlug == "" && folder == "" {
		activator, activationErr := activation.ForProject()
		if activationErr != nil {
			fmt.Fprintln(os.Stderr, "benchmark: provide --provider and --folder, or run inside a bwenv project")
			return 1
		}
		source, resolveErr := activator.Resolve()
		if resolveErr != nil {
			fmt.Fprintln(os.Stderr, "benchmark: provide --provider and --folder, or run inside a bwenv project")
			return 1
		}
		providerSlug, folder, folderID, itemIDs = source.ProviderSlug, source.FolderName, source.FolderID, source.ItemIDs
	}
	if providerSlug == "" || folder == "" {
		fmt.Fprintln(os.Stderr, "benchmark: both --provider and --folder are required")
		return 1
	}
	report, err := benchmark.Benchmark(context.Background(), providerSlug, folder, folderID, itemIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: %v\n", err)
		return 1
	}
	if err := report.Print(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "benchmark: could not write report: %v\n", err)
		return 1
	}
	return 0
}

func runInit(request Request) int {
	if err := ui.RunInitFlow(request.Version, request.Activation); err != nil && !errors.Is(err, ui.ErrCancelled) {
		ui.PrintError("Init failed", err)
		return 1
	}

	return 0
}

func runExport(request Request) int {
	providerSlug, folder, folderID, itemIDs := request.Provider, request.Folder, request.FolderID, request.Items
	projectPath := request.Project
	if projectPath != "" {
		if providerSlug != "" || folder != "" || folderID != "" || len(itemIDs) > 0 {
			ui.PrintError("Invalid flags", fmt.Errorf("--project cannot be combined with providerSlug, folder, or item flags"))
			return 1
		}
		projectConfig, err := project.Load(projectPath)
		if err != nil {
			ui.PrintError("Could not read project config", err)
			return 1
		}
		providerSlug = projectConfig.Provider
		folder = projectConfig.Project.FolderName
		folderID = projectConfig.Project.FolderID
		itemIDs = projectConfig.Project.Items
		if projectConfig.Activation.Disabled {
			if request.Quiet {
				return 0
			}
			ui.PrintError("Export disabled", fmt.Errorf("run bwenv allow or bwenv login to enable this project"))
			return 1
		}
		// Generated scripts can be sourced from another directory. Runtime
		// variable-name metadata belongs to the referenced project, not that cwd.
		info, err := os.Stat(projectPath)
		if err != nil {
			ui.PrintError("Could not locate project", err)
			return 1
		}
		root := projectPath
		if !info.IsDir() {
			root = filepath.Dir(projectPath)
		}
		if err := os.Chdir(root); err != nil {
			ui.PrintError("Could not enter project", err)
			return 1
		}
	}

	if providerSlug == "" || folder == "" {
		ui.PrintError("Missing flags", fmt.Errorf("both --provider and --folder are required"))
		fmt.Fprintln(os.Stderr, "Usage: bwenv export --project <path> | --provider <bitwarden|1password> --folder <name> [--folder-id <id>] [--items 'id1,id2,...']")
		return 1
	}

	quiet := request.Quiet
	exportFunc := export.ExportWithFolderID
	if quiet {
		exportFunc = export.ExportQuiet
	}
	if err := exportFunc(context.Background(), providerSlug, folder, folderID, itemIDs); err != nil {
		// Quiet hooks signal authentication failures to their parent shell.
		if quiet && (errors.Is(err, provider.ErrNotAuthenticated) || errors.Is(err, provider.ErrSessionExpired)) {
			return 2
		}
		// Export already reports other failures on stderr.
		return 1
	}
	return 0
}

func runAllow(request Request) int {
	fi, _ := os.Stdout.Stat()
	isTTY := fi != nil && (fi.Mode()&os.ModeCharDevice) != 0

	if isTTY {
		// Direct invocation without the shell wrapper.
		// Don't print secrets to the terminal — just approve .envrc.
		//
		activator, err := activation.ForProject()
		if err != nil {
			ui.PrintError("Allow failed", err)
			return 1
		}
		source, err := activator.Resolve()
		if err != nil {
			ui.PrintError("Allow failed", err)
			return 1
		}
		if err := activator.Approve(); err != nil {
			ui.PrintError("Allow failed", err)
			return 1
		}
		ui.PrintSuccess(fmt.Sprintf("Project approved · %s / %s", source.ProviderSlug, source.FolderName))
		ui.PrintInfo(`Run 'eval "$(bwenv login)"' (Bash/Zsh) or 'bwenv login | source' (Fish) to load variables.`)
	} else {
		// Pipe mode (via shell wrapper or manual eval) — approve + export.
		_, _, err := export.AllowAndExport()
		if err != nil {
			ui.PrintError("Allow failed", err)
			return 1
		}
	}
	return 0
}

func runDisallow(request Request) int {
	_, err := export.DisallowAndUnset()
	if err != nil {
		ui.PrintError("Disallow failed", err)
		return 1
	}
	ui.PrintSuccess("Project disabled · run bwenv allow to enable")
	return 0
}

func runRemove(request Request) int {
	removed, varNames, err := export.RemoveAndUnset()
	if err != nil {
		ui.PrintError("Remove failed", err)
		return 1
	}

	if removed {
		ui.PrintSuccess(fmt.Sprintf("Project configuration removed · %d variables cleared", len(varNames)))
	} else {
		ui.PrintInfo("No project configuration found")
	}
	return 0
}

func runConfig(request Request) int {
	if err := ui.RunConfigFlow(request.Version); err != nil {
		ui.PrintError("Config failed", err)
		return 1
	}
	return 0
}

func runLogout(request Request) int {
	fi, _ := os.Stdout.Stat()
	if fi != nil && fi.Mode()&os.ModeCharDevice != 0 {
		if err := session.LockAll(context.Background()); err != nil {
			ui.PrintError("Lock failed", err)
			return 1
		}
		ui.PrintInfo(`Logout requested · use the shell wrapper or eval "$(bwenv lock)" to clear this shell`)
		return 0
	}
	if _, err := export.LockAndUnset(context.Background(), request.Shell); err != nil {
		ui.PrintError("Lock incomplete", err)
		return 1
	}
	ui.PrintSuccess("Shell environment cleared · automatic loading locked")
	return 0
}

func runLogin(request Request) int {
	fi, _ := os.Stdout.Stat()
	isTTY := fi != nil && (fi.Mode()&os.ModeCharDevice) != 0

	if isTTY {
		ui.PrintInfo(`Run 'eval "$(bwenv login)"' (Bash/Zsh) or 'bwenv login | source' (Fish) to set the session in this shell.`)
	} else {
		// Pipe mode (via shell wrapper or manual eval) — authenticate + export.
		_, _, err := export.LoginAndExport()
		if err != nil {
			ui.PrintError("Login failed", err)
			return 1
		}
	}
	return 0
}

func runRefresh(request Request) int {
	providerName, synced, err := export.Refresh()
	if err != nil {
		ui.PrintError("Refresh failed", err)
		return 1
	}
	if cfg, err := project.Load(".bwenv.toml"); err == nil && cfg.Activation.Disabled {
		ui.PrintInfo(providerName + " refreshed · project remains disabled")
		return 0
	}
	if activator, aerr := activation.ForProject(); aerr == nil {
		if emitter, ok := activator.(activation.Emitter); ok && emitter.EmitsExports() {
			info, _ := os.Stdout.Stat()
			if info != nil && info.Mode()&os.ModeCharDevice == 0 {
				if _, err := export.ActivateShell(request.Shell); err != nil {
					ui.PrintError("Refresh failed", err)
					return 1
				}
			} else {
				ui.PrintInfo("Use the shell wrapper or evaluate bwenv refresh output to reload variables")
			}
		}
	}
	if synced {
		ui.PrintSuccess(providerName + " synced")
	} else {
		ui.PrintSuccess(providerName + " refreshed")
	}
	return 0
}

func runActivate(request Request) int {
	backend, err := export.ActivateShell(request.Shell)
	if err != nil {
		if errors.Is(err, provider.ErrSessionExpired) || errors.Is(err, provider.ErrNotAuthenticated) {
			ui.PrintWarning("Session locked or expired · run bwenv login")
		} else {
			ui.PrintError("Activation failed", err)
		}
		return 1
	}
	if backend != "shell" {
		ui.PrintSuccess("Project activated · " + backend)
	}
	return 0
}

func runDeactivate(request Request) int {
	varNames, err := export.DeactivateShell(request.Shell)
	if err != nil {
		ui.PrintError("Deactivate failed", err)
		return 1
	}
	ui.PrintSuccess(fmt.Sprintf("Project deactivated · %d variables restored", len(varNames)))
	return 0
}

func runRoot(request Request) int {
	root, err := project.FindRoot(".")
	if err != nil {
		return 1
	}
	if root != "" {
		shellOnly, fingerprint := request.ShellOnly, request.Fingerprint
		if shellOnly {
			cfg, err := project.Load(root)
			if err != nil || cfg.Activation.Mode != "shell" || cfg.Activation.Disabled {
				return 0
			}
		}
		if fingerprint {
			data, err := os.ReadFile(filepath.Join(root, ".bwenv.toml"))
			if err != nil {
				return 1
			}
			// The hook compares this opaque token to detect selection changes
			// even when the user stays in the same project directory.
			fmt.Printf("%s:%x\n", root, sha256.Sum256(data))
			return 0
		}
		fmt.Println(root)
	}
	return 0
}

func runHook(request Request) int {
	name := request.Shell
	if name == "" || name == "auto" {
		name = shell.DetectShell(os.Getenv("SHELL"))
	}

	snippet, err := shell.Hook(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(snippet)
	return 0
}

func runStatus(request Request) int {
	if err := ui.RunStatusFlow(request.Version); err != nil {
		ui.PrintError("Status failed", err)
		return 1
	}
	return 0
}

func runDoctor(request Request) int {
	if err := ui.RunDoctorFlow(request.Version); err != nil {
		return 1
	}
	return 0
}

func runMigrate(request Request) int {
	result, err := direnv.Migrate(request.DryRun)
	if err != nil {
		ui.PrintError("Migration failed", err)
		return 1
	}
	fmt.Print(result)
	return 0
}
