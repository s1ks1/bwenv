// Package shell is the experimental activation backend that loads secrets
// through native shell hooks instead of direnv. It is the v3 default for new
// projects and remains experimental while users validate it.
package shell

import (
	"fmt"
	"path/filepath"
	"strings"
)

// hookMarker identifies the bwenv shell hook in an RC file.
const hookMarker = "# bwenv native shell hook v3 (experimental)"

// SupportedShells lists the shells a hook is available for.
var SupportedShells = []string{"zsh", "bash", "fish"}

// Hook returns the activation hook for the named shell. The hook tracks a
// single active project root and, whenever the prompt fires in a different
// project, deactivates the old project and activates the new one — so nested
// directories and deactivation are handled without external tooling.
func Hook(shellName string) (string, error) {
	switch shellName {
	case "zsh":
		// Guard against re-source duplication: precmd_functions+= appends on
		// every source, running the hook twice per prompt otherwise (PER-45).
		return strings.ReplaceAll(hookPOSIX, "__SHELL__", shellName) + "\n(( ${precmd_functions[(I)_bwenv_prompt_hook]:-0} )) || precmd_functions+=(_bwenv_prompt_hook)\n", nil
	case "bash":
		// Same guard as zsh for PROMPT_COMMAND (PER-45): skip the prepend when
		// the hook is already registered.
		return strings.ReplaceAll(hookPOSIX, "__SHELL__", shellName) + "\ncase \"$PROMPT_COMMAND\" in\n  *_bwenv_prompt_hook*) ;;\n  *) PROMPT_COMMAND=\"_bwenv_prompt_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}\" ;;\nesac\n", nil
	case "fish":
		return hookFish, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", shellName, strings.Join(SupportedShells, ", "))
	}
}

// DetectShell returns the supported shell named by shellPath, defaulting to zsh.
func DetectShell(shellPath string) string {
	switch name := filepath.Base(shellPath); name {
	case "zsh", "bash", "fish":
		return name
	default:
		return "zsh"
	}
}

const hookPOSIX = hookMarker + `
# Loads bwenv secrets when entering a project and clears them when leaving.
_bwenv_active_root="${_bwenv_active_root:-}"
_bwenv_attempted_root="${_bwenv_attempted_root:-}"
_bwenv_attempted_session="${_bwenv_attempted_session:-}"
_bwenv_prompt_hook() {
  local root output
  root="$(command bwenv root --shell-only 2>/dev/null)" || return
  [ -z "$root" ] || [ -f "$root/.bwenv.toml" ] || return 0
  if [ "$root" = "$_bwenv_attempted_root" ] && [ "${BW_SESSION:-}" = "$_bwenv_attempted_session" ]; then
    return 0
  fi
  _bwenv_attempted_root="$root"
  _bwenv_attempted_session="${BW_SESSION:-}"
  if [ "$root" != "$_bwenv_active_root" ]; then
    if [ -n "$_bwenv_active_root" ] && [ -n "${_BWENV_STATE:-}" ]; then
      output="$(command bwenv deactivate --shell "__SHELL__")" || return
      eval "$output"
    fi
    _bwenv_active_root=""
    if [ -n "$root" ]; then
      output="$(command bwenv activate --shell "__SHELL__")" || return
      eval "$output"
      _bwenv_active_root="$root"
    fi
  fi
}
`

const hookFish = hookMarker + `
set -q _bwenv_active_root; or set -g _bwenv_active_root ""
set -q _bwenv_attempted_root; or set -g _bwenv_attempted_root ""
set -q _bwenv_attempted_session; or set -g _bwenv_attempted_session ""
function _bwenv_prompt_hook --on-event fish_prompt
  set -l root (command bwenv root --shell-only 2>/dev/null)
  or return
  if test -n "$root"; and not test -f "$root/.bwenv.toml"
    return
  end
  if test "$root" = "$_bwenv_attempted_root"; and test "$BW_SESSION" = "$_bwenv_attempted_session"
    return
  end
  set -g _bwenv_attempted_root "$root"
  set -g _bwenv_attempted_session "$BW_SESSION"
  if test "$root" != "$_bwenv_active_root"
    if test -n "$_bwenv_active_root"; and set -q _BWENV_STATE
      set -l output (command bwenv deactivate --shell fish | string collect)
      or return
      printf '%s\n' "$output" | source
    end
    set -g _bwenv_active_root ""
    if test -n "$root"
      set -l output (command bwenv activate --shell fish | string collect)
      or return
      printf '%s\n' "$output" | source
      set -g _bwenv_active_root "$root"
    end
  end
end
`
