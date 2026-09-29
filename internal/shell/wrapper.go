package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// wrapperMarker is the unique string used to detect if the bwenv shell wrapper
// function is already installed. It must stay disjoint from the activation
// shell hook marker ("# bwenv shell integration (experimental)") so the two
// installations cannot false-positive on each other (PER-42).
const wrapperMarker = "# bwenv shell wrapper"

// legacyWrapperMarker identifies wrapper installs from before the marker was
// renamed. The full comment line is used so it cannot match the hook marker,
// which shares the "# bwenv shell integration" prefix.
const legacyWrapperMarker = "# bwenv shell integration — enables"

// wrapperBashZsh is the shell function for bash/zsh that wraps bwenv commands.
// Commands that produce shell code (export/unset) are eval'd transparently, so
// "bwenv allow" / "bwenv disallow" / "bwenv remove" / "bwenv login" can modify
// the current shell's environment directly.
const wrapperBashZsh = `
# bwenv shell wrapper — enables seamless secret management
# Commands like allow/disallow/remove/login modify your shell environment directly.
bwenv() {
  case "${1:-}" in
    allow|disallow|deny|remove|clean|export|load|login|auth|activate|deactivate)
      local _bwenv_out
      _bwenv_out="$(command bwenv "$@")"
      local _bwenv_rc=$?
      [ $_bwenv_rc -eq 0 ] && [ -n "$_bwenv_out" ] && eval "$_bwenv_out"
      return $_bwenv_rc
      ;;
    *)
      command bwenv "$@"
      ;;
  esac
}
`

// wrapperFish is the shell function for fish shell.
const wrapperFish = `
# bwenv shell wrapper — enables seamless secret management
function bwenv
  switch $argv[1]
    case allow disallow deny remove clean export load login auth activate deactivate
      set -l _out (command bwenv $argv)
      set -l _rc $status
      if test $_rc -eq 0 -a -n "$_out"
        eval $_out
      end
      return $_rc
    case '*'
      command bwenv $argv
  end
end
`

// InstallWrapper appends the bwenv() shell function to the user's shell RC file.
// The wrapper transparently eval's the output of commands like "bwenv allow",
// "bwenv disallow" and "bwenv remove" so they can modify the current shell's
// environment directly.
//
// Returns (modified bool, filePath string, err error):
//   - modified=true  → wrapper was added to the RC file
//   - modified=false → wrapper already present or error occurred
func InstallWrapper() (modified bool, filePath string, err error) {
	rcPath, err := DetectRC()
	if err != nil {
		return false, "", err
	}

	displayPath := ShortenHomePath(rcPath)

	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, displayPath, fmt.Errorf("could not read %s: %w", displayPath, err)
	}

	if strings.Contains(string(content), wrapperMarker) ||
		strings.Contains(string(content), legacyWrapperMarker) {
		return false, displayPath, nil
	}

	wrapper := wrapperBashZsh
	if filepath.Base(os.Getenv("SHELL")) == "fish" {
		wrapper = wrapperFish
	}

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, displayPath, fmt.Errorf("could not write to %s: %w", displayPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(wrapper); err != nil {
		return false, displayPath, fmt.Errorf("failed to append to %s: %w", displayPath, err)
	}

	return true, displayPath, nil
}
