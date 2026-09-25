package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/s1ks1/bwenv/internal/provider"
)

// RunDoctorFlow prints diagnostics suitable for sharing in an issue report.
func RunDoctorFlow(version string) error {
	PrintBanner(version)
	fmt.Println()
	printStatusSection(E("🩺", "[!]") + " Diagnostics")

	failures := 0
	check := func(ok bool, label, detail string) {
		PrintStatusLine(ok, label, detail)
		if !ok {
			failures++
		}
	}

	PrintInfoLine("System", runtime.GOOS+"/"+runtime.GOARCH)
	shell := filepath.Base(os.Getenv("SHELL"))
	if (shell == "." || shell == "") && runtime.GOOS == "windows" && os.Getenv("PSModulePath") != "" {
		shell = "PowerShell"
	}
	if shell != "." && shell != "" {
		PrintInfoLine("Shell", shell)
	}

	_, direnvErr := exec.LookPath("direnv")
	check(direnvErr == nil, "direnv", statusDetail(direnvErr != nil, "installed", "install from https://direnv.net/"))
	_, hookFound := findDirenvHook()
	check(hookFound, "direnv hook", statusDetail(!hookFound, "configured", "add the hook to your shell RC file"))

	info := checkEnvrcStatus()
	check(info.state == envrcBwenv, ".envrc", envrcDoctorDetail(info.state))
	if info.state == envrcBwenv {
		if info.folderID == "" {
			PrintWarningLine("Folder lookup", "FolderID is missing; run 'bwenv init' to regenerate the fast-path configuration")
		} else {
			PrintStatusLine(true, "Folder lookup", "FolderID fast path configured")
		}

		p, err := provider.Get(info.provider)
		if err != nil {
			check(false, "Provider", "provider configuration is invalid")
		} else {
			available := p.IsAvailable()
			check(available, p.Name(), providerDoctorDetail(available, p.CLICommand()))
			if available {
				authenticated := p.IsAuthenticated()
				check(authenticated, "Provider session", statusDetail(!authenticated, "active", "run 'bwenv login' to authenticate"))
			}
		}
	}

	if info.state != envrcMissing && runtime.GOOS != "windows" {
		private := fileHasPrivatePermissions(".envrc")
		check(private, ".envrc permissions", statusDetail(!private, "private", "run 'chmod 600 .envrc'"))
	}

	fmt.Println()
	if failures > 0 {
		PrintStatusLine(false, "Result", fmt.Sprintf("%d check(s) need attention", failures))
		return fmt.Errorf("%d diagnostic check(s) failed", failures)
	}
	PrintSuccess("Required checks passed")
	return nil
}

func statusDetail(failed bool, success, remedy string) string {
	if failed {
		return remedy
	}
	return success
}

func envrcDoctorDetail(state envrcState) string {
	switch state {
	case envrcBwenv:
		return "bwenv configuration found"
	case envrcPresent:
		return "not managed by bwenv; run 'bwenv init' to configure this project"
	default:
		return "not found; run 'bwenv init' to configure this project"
	}
}

func providerDoctorDetail(available bool, command string) string {
	if available {
		return "CLI installed"
	}
	return "CLI '" + command + "' not found; install the provider CLI"
}

func fileHasPrivatePermissions(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm()&0077 == 0
}
