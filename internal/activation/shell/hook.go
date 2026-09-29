// Package shell is the experimental activation backend that loads secrets
// through native shell hooks instead of direnv. It is not the default: direnv
// remains the stable backend until these hooks are proven.
package shell

import (
	"fmt"
	"path/filepath"
	"strings"
)

// hookMarker identifies the bwenv shell hook in an RC file.
const hookMarker = "# bwenv shell integration (experimental)"

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
		return hookPOSIX + "\n(( ${precmd_functions[(I)_bwenv_prompt_hook]:-0} )) || precmd_functions+=(_bwenv_prompt_hook)\n", nil
	case "bash":
		// Same guard as zsh for PROMPT_COMMAND (PER-45): skip the prepend when
		// the hook is already registered.
		return hookPOSIX + "\ncase \"$PROMPT_COMMAND\" in\n  *_bwenv_prompt_hook*) ;;\n  *) PROMPT_COMMAND=\"_bwenv_prompt_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}\" ;;\nesac\n", nil
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
_bwenv_active_root=""
_bwenv_prompt_hook() {
  local root
  root="$(command bwenv root 2>/dev/null)"
  if [ "$root" != "$_bwenv_active_root" ]; then
    [ -n "$_bwenv_active_root" ] && eval "$(command bwenv deactivate 2>/dev/null)"
    _bwenv_active_root="$root"
    [ -n "$root" ] && eval "$(command bwenv activate 2>/dev/null)"
  fi
}
`

const hookFish = hookMarker + `
set -g _bwenv_active_root ""
function _bwenv_prompt_hook --on-event fish_prompt
  set -l root (command bwenv root 2>/dev/null)
  if test "$root" != "$_bwenv_active_root"
    if test -n "$_bwenv_active_root"
      command bwenv deactivate 2>/dev/null | source
    end
    set -g _bwenv_active_root "$root"
    if test -n "$root"
      command bwenv activate 2>/dev/null | source
    end
  end
end
`
