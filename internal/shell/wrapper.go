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
    allow|disallow|deny|remove|clean|export|load|login|auth|activate|deactivate|refresh)
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
    case allow disallow deny remove clean export load login auth activate deactivate refresh
      set -l _out (command bwenv $argv | string collect)
      set -l _rc $status
      if test $_rc -eq 0 -a -n "$_out"
        eval "$_out"
      end
      return $_rc
    case '*'
      command bwenv $argv
  end
end
`

// loginNotice runs in the parent shell: mise reevaluates env._.source in a
// clean subprocess, so notification state cannot live in that script.
const loginNoticeMarker = "# bwenv login notice"
const loginNoticePOSIX = `
# bwenv login notice
_bwenv_login_notice() {
  if [ "${_BWENV_LOGIN_REQUIRED:-}" != "${_bwenv_login_notified:-}" ]; then
    _bwenv_login_notified="${_BWENV_LOGIN_REQUIRED:-}"
    [ -z "$_bwenv_login_notified" ] || command bwenv login-hint
  fi
}
if [ -n "${ZSH_VERSION:-}" ]; then
  (( ${precmd_functions[(I)_bwenv_login_notice]:-0} )) || precmd_functions+=(_bwenv_login_notice)
else
  case "$PROMPT_COMMAND" in
    *_bwenv_login_notice*) ;;
    *) PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND;}_bwenv_login_notice" ;;
  esac
fi
`
const loginNoticeFish = `
# bwenv login notice
function _bwenv_login_notice --on-event fish_prompt
  set -q _bwenv_login_notified; or set -g _bwenv_login_notified ""
  set -l required ""
  set -q _BWENV_LOGIN_REQUIRED; and set required "$_BWENV_LOGIN_REQUIRED"
  if test "$required" != "$_bwenv_login_notified"
    set -g _bwenv_login_notified "$required"
    if test -n "$required"
      command bwenv login-hint
    end
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

	notice := loginNoticePOSIX
	if filepath.Base(os.Getenv("SHELL")) == "fish" {
		notice = loginNoticeFish
	}
	if strings.Contains(string(content), wrapperMarker) || strings.Contains(string(content), legacyWrapperMarker) {
		updated := string(content)
		for _, old := range []string{
			"allow|disallow|deny|remove|clean|export|load)",
			"allow|disallow|deny|remove|clean|export|load|login|auth|activate|deactivate)",
		} {
			updated = strings.Replace(updated, old, "allow|disallow|deny|remove|clean|export|load|login|auth|activate|deactivate|refresh)", 1)
		}
		if !strings.Contains(updated, loginNoticeMarker) {
			updated += notice
		}
		if updated != string(content) {
			info, err := os.Stat(rcPath)
			if err != nil {
				return false, displayPath, err
			}
			if err := os.WriteFile(rcPath, []byte(updated), info.Mode().Perm()); err != nil {
				return false, displayPath, err
			}
			return true, displayPath, nil
		}
		return false, displayPath, nil
	}

	wrapper := wrapperBashZsh
	if filepath.Base(os.Getenv("SHELL")) == "fish" {
		wrapper = wrapperFish
	}

	if err := os.MkdirAll(filepath.Dir(rcPath), 0755); err != nil {
		return false, displayPath, err
	}
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, displayPath, fmt.Errorf("could not write to %s: %w", displayPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(wrapper + notice); err != nil {
		return false, displayPath, fmt.Errorf("failed to append to %s: %w", displayPath, err)
	}

	return true, displayPath, nil
}
