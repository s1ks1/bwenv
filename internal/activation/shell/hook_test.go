package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHookForEachShell(t *testing.T) {
	for _, name := range SupportedShells {
		snippet, err := Hook(name)
		if err != nil {
			t.Fatalf("Hook(%q) error: %v", name, err)
		}
		for _, want := range []string{hookMarker, "bwenv root", "bwenv activate", "bwenv deactivate"} {
			if !strings.Contains(snippet, want) {
				t.Fatalf("Hook(%q) missing %q:\n%s", name, want, snippet)
			}
		}
	}
}

func TestHookRejectsUnsupportedShell(t *testing.T) {
	if _, err := Hook("tcsh"); err == nil {
		t.Fatal("expected an error for an unsupported shell")
	}
}

func TestHookIgnoresLegacyCLIHelpOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture; exercised on Linux/macOS")
	}
	for _, name := range []string{"bash", "zsh"} {
		t.Run(name, func(t *testing.T) {
			path, err := exec.LookPath(name)
			if err != nil {
				t.Skip(name + " unavailable")
			}
			dir := t.TempDir()
			fake := "#!/bin/sh\nprintf '╭─ old bwenv help ─╮\\nSetup:\\n'\n"
			if err := os.WriteFile(filepath.Join(dir, "bwenv"), []byte(fake), 0755); err != nil {
				t.Fatal(err)
			}
			hook, err := Hook(name)
			if err != nil {
				t.Fatal(err)
			}
			script := `export PATH="$1"` + "\n" + hook + "\n_bwenv_prompt_hook\n"
			cmd := exec.Command(path, "-c", script, name, dir)
			cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), "ZDOTDIR="+t.TempDir())
			output, err := cmd.CombinedOutput()
			if err != nil || len(output) != 0 {
				t.Fatalf("legacy CLI produced startup errors: %v\n%s", err, output)
			}
		})
	}
}

func TestDetectShell(t *testing.T) {
	cases := map[string]string{
		"/bin/zsh":      "zsh",
		"/usr/bin/bash": "bash",
		"/usr/bin/fish": "fish",
		"":              "zsh",
		"/bin/sh":       "zsh",
	}
	for in, want := range cases {
		if got := DetectShell(in); got != want {
			t.Errorf("DetectShell(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBashHookIdempotentOnResource is the regression test for PER-45: sourcing
// the RC file twice used to prepend _bwenv_prompt_hook to PROMPT_COMMAND again,
// running the hook twice per prompt.
func TestBashHookIdempotentOnResource(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	snippet, err := Hook("bash")
	if err != nil {
		t.Fatalf("Hook(bash): %v", err)
	}
	hookFile := filepath.Join(t.TempDir(), "hook.sh")
	if err := os.WriteFile(hookFile, []byte(snippet), 0644); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	script := `PROMPT_COMMAND="$2"
source "$1"
source "$1"
printf '%s' "$PROMPT_COMMAND"`

	for _, start := range []string{"", "existing_cmd"} {
		out, err := exec.Command(bash, "-c", script, "bash", hookFile, start).CombinedOutput()
		if err != nil {
			t.Fatalf("bash -c (start=%q): %v\n%s", start, err, out)
		}
		got := string(out)
		if n := strings.Count(got, "_bwenv_prompt_hook"); n != 1 {
			t.Errorf("PROMPT_COMMAND after double source (start=%q): %d hook entries, want 1: %q", start, n, got)
		}
		if start != "" && !strings.Contains(got, start) {
			t.Errorf("PROMPT_COMMAND lost pre-existing entries (start=%q): %q", start, got)
		}
	}
}

// TestZshHookIdempotentOnResource covers the same PER-45 bug class for zsh:
// precmd_functions+=(_bwenv_prompt_hook) duplicated on every re-source.
func TestZshHookIdempotentOnResource(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not available")
	}
	snippet, err := Hook("zsh")
	if err != nil {
		t.Fatalf("Hook(zsh): %v", err)
	}
	hookFile := filepath.Join(t.TempDir(), "hook.zsh")
	if err := os.WriteFile(hookFile, []byte(snippet), 0644); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	script := `precmd_functions=(existing_fn)
source "$1"
source "$1"
print -r -- "${precmd_functions}"`

	out, err := exec.Command(zsh, "-c", script, "zsh", hookFile).CombinedOutput()
	if err != nil {
		t.Fatalf("zsh -c: %v\n%s", err, out)
	}
	got := string(out)
	if n := strings.Count(got, "_bwenv_prompt_hook"); n != 1 {
		t.Errorf("precmd_functions after double source: %d hook entries, want 1: %q", n, got)
	}
	if !strings.Contains(got, "existing_fn") {
		t.Errorf("precmd_functions lost pre-existing entries: %q", got)
	}
}

func TestFishHookDoesNotApplyFailedActivationOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish unavailable")
	}
	bin := t.TempDir()
	fake := "#!/bin/sh\ncase \"$1\" in\nroot) printf '/fixture:fingerprint\\n' ;;\nactivate) printf 'set -gx API_KEY partial-output\\n'; exit 9 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "bwenv"), []byte(fake), 0755); err != nil {
		t.Fatal(err)
	}
	hook, _ := Hook("fish")
	script := hook + "\nset -gx API_KEY original\n_bwenv_prompt_hook\ntest \"$API_KEY\" = original; and test -z \"$_bwenv_active_root\"\n"
	command := exec.Command(fish, "-c", script)
	command.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if result, err := command.CombinedOutput(); err != nil {
		t.Fatalf("failed activation was evaluated: %v %s", err, result)
	}
}
