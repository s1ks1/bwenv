// Package check provides diagnostic functions for the "bwenv test" command.
// It verifies that all required dependencies (CLI tools, direnv hooks, etc.)
// are installed and properly configured on the user's system, then prints
// a styled status report using the ui package.
package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/s1ks1/bwenv/internal/provider"
	"github.com/s1ks1/bwenv/internal/ui"
)

// RunDiagnostics checks all dependencies and configuration, then prints
// a comprehensive status report to the terminal. This is the handler
// for the "bwenv test" command.
func RunDiagnostics(version string) {
	ui.PrintBanner(version)

	fmt.Println()
	ui.PrintSection("🖥  System Information")
	printSystemInfo()

	fmt.Println()
	ui.PrintSection("📦 Core Dependencies")
	checkCoreDependencies()

	fmt.Println()
	ui.PrintSection("🔑 Secret Providers")
	checkProviders()

	fmt.Println()
	ui.PrintSection("⚙  Direnv Configuration")
	checkDirenvSetup()

	fmt.Println()
	ui.PrintSection("🌍 Environment")
	checkEnvironment()

	fmt.Println()
}

// printSystemInfo displays basic system information (OS, architecture, shell).
// This helps when troubleshooting cross-platform issues.
func printSystemInfo() {
	// Operating system and architecture.
	ui.PrintKeyValue("OS", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))

	// Detect the user's default shell.
	shell := detectShell()
	if shell != "" {
		ui.PrintKeyValue("Shell", shell)
	} else {
		ui.PrintKeyValue("Shell", "(unknown)")
	}

	// Show the current working directory.
	if cwd, err := os.Getwd(); err == nil {
		ui.PrintKeyValue("Working Dir", cwd)
	}
}

// checkCoreDependencies verifies that essential tools (direnv) are installed.
// These are needed regardless of which secret provider is used.
func checkCoreDependencies() {
	// Check for direnv — this is required for .envrc loading.
	checkBinary("direnv", "https://direnv.net/", true)
}

// checkProviders iterates over all registered secret providers and reports
// whether each one's CLI tool is installed and accessible.
func checkProviders() {
	allProviders := provider.All()

	if len(allProviders) == 0 {
		ui.PrintWarningLine("No providers registered", "this is likely a bug")
		return
	}

	// Track how many providers are available so we can warn if none are.
	availableCount := 0

	for _, p := range allProviders {
		cliCmd := p.CLICommand()
		isAvailable := p.IsAvailable()

		if isAvailable {
			availableCount++

			// Get the version string from the CLI tool if possible.
			ver := getCommandVersion(cliCmd)
			detail := ""
			if ver != "" {
				detail = fmt.Sprintf("(%s v%s)", cliCmd, ver)
			} else {
				detail = fmt.Sprintf("(%s)", cliCmd)
			}
			ui.PrintStatusLine(true, p.Name(), detail)

			// For available providers, also check if they are authenticated.
			if p.IsAuthenticated() {
				ui.PrintStatusLine(true, fmt.Sprintf("  %s session", p.Name()), "🟢 active")
			} else {
				ui.PrintInfoLine(fmt.Sprintf("  %s session", p.Name()), "not active — will prompt on use")
			}
		} else {
			ui.PrintStatusLine(false, p.Name(), fmt.Sprintf("'%s' not found in PATH — install to enable", cliCmd))
		}
	}

	// Show a warning if no providers are usable.
	if availableCount == 0 {
		fmt.Println()
		ui.PrintWarning("⚠  No secret providers available — install at least one CLI tool to use bwenv")
	} else {
		fmt.Println()
		ui.PrintInfo(fmt.Sprintf("%d provider(s) ready to use", availableCount))
	}
}

// checkDirenvSetup verifies that direnv is properly hooked into the user's shell.
// It checks common shell config files for the direnv hook line.
func checkDirenvSetup() {
	// First check if direnv is installed at all.
	if _, err := exec.LookPath("direnv"); err != nil {
		ui.PrintStatusLine(false, "direnv", "not installed — skipping hook check")
		return
	}

	ui.PrintStatusLine(true, "direnv installed", "")

	// Check for the direnv hook in shell config files.
	// We look for common patterns in the usual RC files.
	hookFound := false
	shellConfigs := getShellConfigPaths()

	for _, rc := range shellConfigs {
		if _, err := os.Stat(rc); os.IsNotExist(err) {
			continue
		}

		content, err := os.ReadFile(rc)
		if err != nil {
			continue
		}

		// Look for common direnv hook patterns.
		contentStr := string(content)
		if strings.Contains(contentStr, "direnv hook") ||
			strings.Contains(contentStr, "direnv export") {
			shortPath := shortenHomePath(rc)
			ui.PrintStatusLine(true, "direnv hook", fmt.Sprintf("found in %s ✓", shortPath))
			hookFound = true
			break
		}
	}

	if !hookFound {
		ui.PrintWarningLine("direnv hook not found in shell config",
			"add it to your shell RC file")

		// Print a helpful hint about how to add the hook.
		shell := detectShell()
		switch {
		case strings.Contains(shell, "bash"):
			ui.PrintInfoLine("  Add to ~/.bashrc:", `eval "$(direnv hook bash)"`)
		case strings.Contains(shell, "zsh"):
			ui.PrintInfoLine("  Add to ~/.zshrc:", `eval "$(direnv hook zsh)"`)
		case strings.Contains(shell, "fish"):
			ui.PrintInfoLine("  Add to config.fish:", "direnv hook fish | source")
		default:
			ui.PrintInfoLine("  See:", "https://direnv.net/docs/hook.html")
		}
	}

	// Check if there's an .envrc in the current directory.
	if _, err := os.Stat(".envrc"); err == nil {
		ui.PrintStatusLine(true, ".envrc exists", "📄 in current directory")

		// Try to read the first few lines to identify if it was generated by bwenv.
		content, err := os.ReadFile(".envrc")
		if err == nil && strings.Contains(string(content), "bwenv") {
			ui.PrintStatusLine(true, ".envrc type", "generated by bwenv ✓")

			// Check if it uses the new DIRENV_LOG_FORMAT suppression.
			if !strings.Contains(string(content), "DIRENV_LOG_FORMAT") {
				ui.PrintWarningLine(".envrc outdated", "re-run 'bwenv init' to get the improved format")
			}
		}
	} else {
		ui.PrintInfoLine(".envrc", "not found in current directory — run 'bwenv init' to create one")
	}
}

// checkEnvironment verifies environment variables that bwenv depends on.
func checkEnvironment() {
	// Check BW_SESSION for Bitwarden.
	if session := os.Getenv("BW_SESSION"); session != "" {
		ui.PrintStatusLine(true, "BW_SESSION", "🔐 set")
	} else {
		ui.PrintInfoLine("BW_SESSION", "not set — will be requested when using Bitwarden")
	}

	// Check OP_SERVICE_ACCOUNT_TOKEN for 1Password service accounts.
	if token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"); token != "" {
		ui.PrintStatusLine(true, "OP_SERVICE_ACCOUNT_TOKEN", "🔐 set")
	} else {
		ui.PrintInfoLine("OP_SERVICE_ACCOUNT_TOKEN", "not set — optional, for 1Password service accounts")
	}

	// Check if DIRENV_WARN_TIMEOUT is set (we set it in generated .envrc files).
	if timeout := os.Getenv("DIRENV_WARN_TIMEOUT"); timeout != "" {
		ui.PrintStatusLine(true, "DIRENV_WARN_TIMEOUT", timeout)
	}

	// Check DIRENV_LOG_FORMAT — bwenv sets this to "" to silence direnv noise.
	if logFmt, ok := os.LookupEnv("DIRENV_LOG_FORMAT"); ok && logFmt == "" {
		ui.PrintStatusLine(true, "DIRENV_LOG_FORMAT", "silenced (bwenv manages output)")
	}
}

// checkBinary verifies that a CLI tool is installed and reports its status.
// If required is true and the tool is missing, it prints a more urgent message.
func checkBinary(name string, installURL string, required bool) {
	path, err := exec.LookPath(name)
	if err != nil {
		if required {
			ui.PrintStatusLine(false, name, fmt.Sprintf("not found — install from %s", installURL))
		} else {
			ui.PrintInfoLine(name, fmt.Sprintf("not installed (optional) — %s", installURL))
		}
		return
	}

	// Get the version string if possible.
	ver := getCommandVersion(name)
	detail := path
	if ver != "" {
		detail = fmt.Sprintf("v%s (%s)", ver, path)
	}

	ui.PrintStatusLine(true, name, detail)
}

// getCommandVersion tries to get the version string from a CLI tool.
// It attempts common version flags (--version, version) and returns
// the first line of output, or an empty string if all attempts fail.
func getCommandVersion(name string) string {
	// Try different version flags that various tools use.
	versionFlags := [][]string{
		{"--version"},
		{"version"},
		{"-v"},
	}

	for _, flags := range versionFlags {
		cmd := exec.Command(name, flags...)
		out, err := cmd.Output()
		if err != nil {
			continue
		}

		// Take the first line and clean it up.
		version := strings.TrimSpace(string(out))
		if idx := strings.IndexByte(version, '\n'); idx >= 0 {
			version = version[:idx]
		}

		// Strip common prefixes like "v", the tool name, etc.
		version = strings.TrimSpace(version)
		version = strings.TrimPrefix(version, name)
		version = strings.TrimSpace(version)
		version = strings.TrimPrefix(version, "v")
		version = strings.TrimPrefix(version, "version")
		version = strings.TrimSpace(version)

		if version != "" {
			return version
		}
	}

	return ""
}

// detectShell returns the name of the user's default shell.
// It checks the SHELL environment variable on Unix systems
// and falls back to common defaults on Windows.
func detectShell() string {
	// On Unix-like systems, SHELL is the standard env var.
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}

	// On Windows, check for common shell indicators.
	if runtime.GOOS == "windows" {
		// Check if running in PowerShell.
		if os.Getenv("PSModulePath") != "" {
			return "powershell"
		}
		// Check for Git Bash / MSYS2.
		if os.Getenv("MSYSTEM") != "" {
			return "bash (Git Bash)"
		}
		return "cmd"
	}

	return ""
}

// getShellConfigPaths returns a list of common shell configuration file paths
// to check for direnv hooks. The list varies by OS and detected shell.
func getShellConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Collect all common RC files — we check multiple because users
	// might use different shells or have migrated their config.
	paths := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".zprofile"),
		filepath.Join(home, ".config", "fish", "config.fish"),
		filepath.Join(home, ".profile"),
	}

	// On macOS, also check .zprofile which is the default for newer macOS.
	if runtime.GOOS == "darwin" {
		paths = append(paths, filepath.Join(home, ".zlogin"))
	}

	return paths
}

// shortenHomePath replaces the user's home directory prefix with "~"
// for more compact and readable output in status messages.
func shortenHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
