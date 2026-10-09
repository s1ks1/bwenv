package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/project"
	runshell "github.com/s1ks1/bwenv/v3/internal/shell"
)

func TestActivationIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native shell integration requires Bash/Zsh/Fish")
	}
	// Capture an installed Node before per-test HOME isolation hides mise's data.
	var miseNode string
	if mise, err := exec.LookPath("mise"); err == nil {
		if result, err := exec.Command(mise, "which", "node").Output(); err == nil {
			miseNode = strings.TrimSpace(string(result))
		}
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	bin := t.TempDir()
	for target, source := range map[string]string{"bwenv": "./cmd/bwenv", "bw": "./tests/fixtures/fakecli"} {
		if output, err := exec.Command("go", "build", "-o", filepath.Join(bin, target), source).CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", target, err, output)
		}
	}
	if err := os.Symlink(filepath.Join(bin, "bw"), filepath.Join(bin, "op")); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	first, second := filepath.Join(base, "first project"), filepath.Join(base, "second")
	for dir, item := range map[string]string{first: "item-1", second: "item-2"} {
		if err := os.MkdirAll(filepath.Join(dir, "nested"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := project.Write(filepath.Join(dir, ".bwenv.toml"), project.Config{Version: 1, Provider: "bitwarden", Project: project.Metadata{FolderName: "Fixture", FolderID: "folder-1", Items: []string{item}}, Activation: project.Activation{Mode: "shell"}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"bash", "zsh", "fish"} {
		t.Run(name, func(t *testing.T) {
			path, err := exec.LookPath(name)
			if err != nil {
				t.Skip(name + " not installed")
			}
			log := filepath.Join(t.TempDir(), "calls.log")
			script := `set -e
export PATH="$_BWENV_TEST_PATH"
eval "$(command bwenv hook "$1")"
cd "$2/nested"
_bwenv_prompt_hook
[ "$API_KEY" = fake-secret-value ]
[ "${DB_URL+x}" != x ]
cd "$2"
eval "$(command bwenv hook "$1")"
_bwenv_prompt_hook
[ "$API_KEY" = fake-secret-value ]
cd "$3"
_bwenv_prompt_hook
[ "$API_KEY" = "$ORIGINAL" ]
[ "$DB_URL" = fake-database-value ]
cd "$4"
_bwenv_prompt_hook
[ "$API_KEY" = "$ORIGINAL" ]
[ "${DB_URL+x}" != x ]
[ "${_BWENV_STATE+x}" != x ]
cd "$2"
export BWENV_FAKE_SCENARIO=expired
_bwenv_prompt_hook || true
_bwenv_prompt_hook
_bwenv_prompt_hook
[ "$API_KEY" = "$ORIGINAL" ]
unset BWENV_FAKE_SCENARIO
export BW_SESSION=renewed-session
_bwenv_prompt_hook
[ "$API_KEY" = fake-secret-value ]
cd "$4"
_bwenv_prompt_hook
[ "$API_KEY" = "$ORIGINAL" ]`
			if name == "fish" {
				script = `set -gx PATH (string split : "$_BWENV_TEST_PATH")
command bwenv hook fish | source
cd "$argv[2]/nested"
_bwenv_prompt_hook
test "$API_KEY" = fake-secret-value; or exit 1
set -q DB_URL; and exit 2
cd "$argv[2]"
command bwenv hook fish | source
_bwenv_prompt_hook
test "$API_KEY" = fake-secret-value; or exit 3
cd "$argv[3]"
_bwenv_prompt_hook
test "$API_KEY" = "$ORIGINAL"; or exit 4
test "$DB_URL" = fake-database-value; or exit 5
cd "$argv[4]"
_bwenv_prompt_hook
test "$API_KEY" = "$ORIGINAL"; or exit 6
set -q DB_URL; and exit 7
set -q _BWENV_STATE; and exit 8
exit 0`
			}
			cmd := exec.Command(path, "-c", script, name, name, first, second, base)
			if name == "fish" {
				cmd = exec.Command(path, "-c", script, name, first, second, base)
			}
			original := "original ' \\ $(touch forbidden)\nsecond line"
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "_BWENV_TEST_PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "HOME="+t.TempDir(), "ZDOTDIR="+t.TempDir(), "SHELL="+path, "BW_SESSION=fixture-session", "XDG_CONFIG_HOME="+t.TempDir(), "BWENV_FAKE_LOG="+log, "API_KEY="+original, "ORIGINAL="+original)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("hook lifecycle: %v\n%s", err, output)
			}
			if name != "fish" && strings.Count(string(output), "Activation failed:") != 1 {
				t.Fatalf("expected one concise authentication warning: %s", output)
			}
			if strings.Contains(string(output), "Authentication failed") || strings.Contains(string(output), "bwenv export error") {
				t.Fatal("duplicate or boxed activation error")
			}
			calls, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			want := 4 // first, second, failed activation, retry; nested/re-source perform no fetch
			if name == "fish" {
				want = 2
			}
			if got := strings.Count(string(calls), "list items"); got != want {
				t.Fatalf("provider calls = %d, want %d: %s", got, want, calls)
			}
		})
	}
	t.Run("1password selection change in same directory", func(t *testing.T) {
		for _, name := range []string{"bash", "zsh", "fish"} {
			t.Run(name, func(t *testing.T) {
				path, err := exec.LookPath(name)
				if err != nil {
					t.Skip(name + " not installed")
				}
				root := t.TempDir()
				cfg := project.Config{Version: 1, Provider: "1password", Project: project.Metadata{FolderName: "Fixture", FolderID: "vault-1", Items: []string{"item-1"}}, Activation: project.Activation{Mode: "shell"}}
				if err := project.Write(filepath.Join(root, ".bwenv.toml"), cfg); err != nil {
					t.Fatal(err)
				}
				cfg.Project.Items = []string{"item-2"}
				if err := project.Write(filepath.Join(root, "next.toml"), cfg); err != nil {
					t.Fatal(err)
				}
				log := filepath.Join(t.TempDir(), "calls.log")
				script := `set -e
export PATH="$_BWENV_TEST_PATH"
unset BW_SESSION _BWENV_STATE OLD_ONLY NEW_ONLY
eval "$(command bwenv hook "$1")"
_bwenv_prompt_hook
[ "$API_KEY" = fake-secret-value ]
[ "$OLD_ONLY" = old-only-value ]
cp next.toml .bwenv.toml
_bwenv_prompt_hook
[ "$API_KEY" = new-note-value ]
[ "${OLD_ONLY+x}" != x ]
[ "$NEW_ONLY" = new-only-value ]
_bwenv_prompt_hook
cd ..
_bwenv_prompt_hook
[ "$API_KEY" = original ]
[ "${OLD_ONLY+x}" != x ]
[ "${NEW_ONLY+x}" != x ]`
				if name == "fish" {
					script = `set -gx PATH (string split : "$_BWENV_TEST_PATH")
set -e BW_SESSION _BWENV_STATE OLD_ONLY NEW_ONLY
command bwenv hook fish | source
_bwenv_prompt_hook
test "$API_KEY" = fake-secret-value; or exit 1
test "$OLD_ONLY" = old-only-value; or exit 2
cp next.toml .bwenv.toml
_bwenv_prompt_hook
test "$API_KEY" = new-note-value; or exit 3
set -q OLD_ONLY; and exit 4
test "$NEW_ONLY" = new-only-value; or exit 5
_bwenv_prompt_hook
cd ..
_bwenv_prompt_hook
test "$API_KEY" = original; or exit 6
set -q OLD_ONLY; and exit 7
set -q NEW_ONLY; and exit 8
exit 0`
				}
				cmd := exec.Command(path, "-c", script, name, name)
				cmd.Dir = root
				cmd.Env = append(os.Environ(), "_BWENV_TEST_PATH="+bin+":"+os.Getenv("PATH"), "HOME="+t.TempDir(), "ZDOTDIR="+t.TempDir(), "XDG_CONFIG_HOME="+t.TempDir(), "SHELL="+path, "BWENV_FAKE_LOG="+log, "API_KEY=original", "BWENV_FAKE_SCENARIO=selection-change")
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("selection change: %v\n%s", err, output)
				}
				calls, err := os.ReadFile(log)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(calls), "op item get") != 2 || strings.Contains(string(calls), "op item list") {
					t.Fatalf("expected only two selected-note fetches: %s", calls)
				}
			})
		}
	})
	t.Run("service account allow and logout", func(t *testing.T) {
		for _, name := range []string{"bash", "zsh"} {
			t.Run(name, func(t *testing.T) {
				path, err := exec.LookPath(name)
				if err != nil {
					t.Skip(name + " unavailable")
				}
				t.Chdir(t.TempDir())
				backend, _ := activation.Get("shell")
				if err := backend.Install(activation.Config{ProviderSlug: "1password", FolderName: "Fixture", FolderID: "vault-1", ItemIDs: []string{"item-1"}}); err != nil {
					t.Fatal(err)
				}
				home := t.TempDir()
				t.Setenv("HOME", home)
				t.Setenv("SHELL", path)
				if _, _, err := runshell.InstallWrapper(); err != nil {
					t.Fatal(err)
				}
				rc := ".bashrc"
				if name == "zsh" {
					rc = ".zshrc"
				}
				cmd := exec.Command(path, "-c", `set -e
export PATH="$_BWENV_TEST_PATH"
source "$1/$2"
export API_KEY=original OP_SERVICE_ACCOUNT_TOKEN=fixture-token
bwenv allow
[ "$API_KEY" = fake-secret-value ]
bwenv logout
[ "$API_KEY" = original ]
[ "${OP_SERVICE_ACCOUNT_TOKEN+x}" != x ]
[ "$_BWENV_LOCKED" = 1 ]`, name, home, rc)
				cmd.Env = append(os.Environ(), "_BWENV_TEST_PATH="+bin+":"+os.Getenv("PATH"), "XDG_CONFIG_HOME="+t.TempDir(), "BWENV_FAKE_SCENARIO=service-account")
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("service account lifecycle: %v\n%s", err, output)
				}
			})
		}
	})
	t.Run("persistent disable and shell lock", func(t *testing.T) {
		for _, name := range []string{"bash", "zsh"} {
			for _, slug := range []string{"bitwarden", "1password"} {
				t.Run(name+"/"+slug, func(t *testing.T) {
					path, err := exec.LookPath(name)
					if err != nil {
						t.Skip(name + " unavailable")
					}
					root, home := t.TempDir(), t.TempDir()
					cfg := project.Config{Version: 1, Provider: slug, Project: project.Metadata{FolderName: "Fixture", FolderID: "folder-1", Items: []string{"item-1"}}, Activation: project.Activation{Mode: "shell"}}
					if slug == "1password" {
						cfg.Project.FolderID = "vault-1"
					}
					if err := project.Write(filepath.Join(root, ".bwenv.toml"), cfg); err != nil {
						t.Fatal(err)
					}
					t.Setenv("HOME", home)
					t.Setenv("USERPROFILE", home)
					t.Setenv("SHELL", path)
					if _, _, err := runshell.InstallWrapper(); err != nil {
						t.Fatal(err)
					}
					log := filepath.Join(t.TempDir(), "calls.log")
					script := `set -e
export PATH="$_BWENV_TEST_PATH"
unset _BWENV_STATE _BWENV_LOCKED OLD_ONLY
source "$2/.$1rc"
eval "$(command bwenv hook "$1")"
cd "$3"
_bwenv_prompt_hook
[ "$API_KEY" = fake-secret-value ]
bwenv disallow
[ "$API_KEY" = original ]
bwenv refresh
[ "$API_KEY" = original ]
_bwenv_prompt_hook
cd ..
_bwenv_prompt_hook
cd "$3"
_bwenv_prompt_hook
[ "$API_KEY" = original ]
# A new shell must also honor the persisted disabled setting.
"$SHELL" -c 'eval "$(command bwenv hook "$1")"; _bwenv_prompt_hook; [ "$API_KEY" = original ]' "$1" "$1"
bwenv lock
[ "$API_KEY" = original ]
bwenv allow
[ "$API_KEY" = fake-secret-value ]
_bwenv_prompt_hook
export OP_SESSION_fixture=fixture-token OP_SERVICE_ACCOUNT_TOKEN=fixture-token
bwenv lock
[ "$API_KEY" = original ]
[ "${BW_SESSION+x}" != x ]
[ "${OP_SESSION_fixture+x}" != x ]
[ "${OP_SERVICE_ACCOUNT_TOKEN+x}" != x ]
[ "${_BWENV_STATE+x}" != x ]
[ "$_BWENV_LOCKED" = 1 ]
before=$(wc -l < "$BWENV_FAKE_LOG")
_bwenv_prompt_hook || true
_bwenv_prompt_hook
cd ..
_bwenv_prompt_hook
cd "$3"
_bwenv_prompt_hook || true
_bwenv_prompt_hook
[ "$API_KEY" = original ]
[ "$(wc -l < "$BWENV_FAKE_LOG")" = "$before" ]
bwenv login
[ "$API_KEY" = fake-secret-value ]
[ "${_BWENV_LOCKED+x}" != x ]
export BWENV_FAKE_SCENARIO=lock-error
if bwenv logout; then exit 9; fi
[ "$API_KEY" = original ]
[ "${BW_SESSION+x}" != x ]
[ "${_BWENV_STATE+x}" != x ]
[ "$_BWENV_LOCKED" = 1 ]
export BWENV_FAKE_SCENARIO=provider-error
if bwenv login; then exit 10; fi
[ "$API_KEY" = original ]
[ "$_BWENV_LOCKED" = 1 ]
unset BWENV_FAKE_SCENARIO
bwenv login
[ "$API_KEY" = fake-secret-value ]`
					cmd := exec.Command(path, "-c", script, name, name, home, root)
					cmd.Env = append(os.Environ(), "_BWENV_TEST_PATH="+bin+":"+os.Getenv("PATH"), "BW_SESSION=fixture-session", "XDG_CONFIG_HOME="+t.TempDir(), "ZDOTDIR="+home, "API_KEY=original", "BWENV_FAKE_LOG="+log)
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("disable/lock lifecycle: %v\n%s", err, output)
					}
				})
			}
		}
	})
	t.Run("plain login", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("SHELL", "/bin/bash")
		if _, _, err := runshell.InstallWrapper(); err != nil {
			t.Fatal(err)
		}
		bash, err := exec.LookPath("bash")
		if err != nil {
			t.Skip("bash unavailable")
		}
		script := `set -e
export PATH="$_BWENV_TEST_PATH"
unset BW_SESSION
cd "$1/nested"
source "$2/.bashrc"
bwenv login
[ "$API_KEY" = fake-secret-value ]
[ "$BW_SESSION" = fake-session ]
[ "${DB_URL+x}" != x ]`
		cmd := exec.Command(bash, "-c", script, "bash", first, home)
		cmd.Env = append(os.Environ(), "_BWENV_TEST_PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "XDG_CONFIG_HOME="+t.TempDir())
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("plain login did not set shell variables: %v\n%s", err, output)
		}
	})
	t.Run("nested daily commands", func(t *testing.T) {
		root, home := t.TempDir(), t.TempDir()
		if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
			t.Fatal(err)
		}
		cfg, err := project.Load(filepath.Join(first, ".bwenv.toml"))
		if err != nil {
			t.Fatal(err)
		}
		if err := project.Write(filepath.Join(root, ".bwenv.toml"), cfg); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("SHELL", "/bin/bash")
		if _, _, err := runshell.InstallWrapper(); err != nil {
			t.Fatal(err)
		}
		script := `set -e
export PATH="$_BWENV_TEST_PATH"
source "$2/.bashrc"
cd "$1/nested"
export API_KEY=original
bwenv login
[ "$API_KEY" = fake-secret-value ]
bwenv refresh
[ "$API_KEY" = fake-secret-value ]
bwenv disallow
[ "$API_KEY" = original ]
bwenv disallow
[ "$API_KEY" = original ]
bwenv remove
[ ! -e "$1/.bwenv.toml" ]
[ ! -e "$1/.bwenv_vars" ]
[ "$API_KEY" = original ]`
		cmd := exec.Command("bash", "-c", script, "bash", root, home)
		cmd.Env = append(os.Environ(), "_BWENV_TEST_PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "BW_SESSION=fixture-session", "XDG_CONFIG_HOME="+t.TempDir())
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("nested daily commands: %v\n%s", err, output)
		}
	})
	t.Run("direnv login output", func(t *testing.T) {
		if _, err := exec.LookPath("direnv"); err != nil {
			t.Skip("direnv unavailable")
		}
		t.Chdir(t.TempDir())
		backend, _ := activation.Get("direnv")
		if err := backend.Install(activation.Config{ProviderSlug: "bitwarden", FolderName: "Fixture", FolderID: "folder-1", ItemIDs: []string{"item-1"}}); err != nil {
			t.Fatal(err)
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("SHELL", "/bin/bash")
		if _, _, err := runshell.InstallWrapper(); err != nil {
			t.Fatal(err)
		}
		script := `set -e
export PATH="$_BWENV_TEST_PATH"
source "$1/.bashrc"
unset BW_SESSION API_KEY
bwenv login
eval "$(direnv export bash)"
[ "$API_KEY" = fake-secret-value ]
bwenv login
eval "$(direnv export bash)"
[ "$API_KEY" = fake-secret-value ]`
		cmd := exec.Command("bash", "-c", script, "bash", home)
		cmd.Env = append(os.Environ(), "_BWENV_TEST_PATH="+bin+":"+os.Getenv("PATH"), "XDG_CONFIG_HOME="+t.TempDir())
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("direnv login: %v\n%s", err, output)
		}
		if strings.Count(string(output), "1 variable loaded") != 2 {
			t.Fatalf("expected one summary per login: %s", output)
		}
		if strings.Contains(string(output), "fake-secret-value") || strings.Contains(string(output), "direnv:") {
			t.Fatalf("noisy or unsafe output: %s", output)
		}
	})
	t.Run("approval failure", func(t *testing.T) {
		t.Chdir(t.TempDir())
		backend, _ := activation.Get("direnv")
		if err := backend.Install(activation.Config{ProviderSlug: "bitwarden", FolderName: "Fixture", FolderID: "folder-1", ItemIDs: []string{"item-1"}}); err != nil {
			t.Fatal(err)
		}
		fakeDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(fakeDir, "direnv"), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatal(err)
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("SHELL", "/bin/bash")
		if _, _, err := runshell.InstallWrapper(); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"login", "allow"} {
			direct := exec.Command(filepath.Join(bin, "bwenv"), name)
			direct.Env = append(os.Environ(), "PATH="+fakeDir+":"+bin+":"+os.Getenv("PATH"), "BW_SESSION=fixture-session", "XDG_CONFIG_HOME="+t.TempDir())
			var stdout, stderr bytes.Buffer
			direct.Stdout, direct.Stderr = &stdout, &stderr
			if err := direct.Run(); err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "approval failed") || strings.Contains(stderr.String(), "loaded") {
				t.Fatalf("false success for %s: stdout=%q stderr=%q", name, stdout.String(), stderr.String())
			}
			wrapper := exec.Command("bash", "-c", `source "$1/.bashrc"; export API_KEY=original; bwenv "$2"; result=$?; [ "$result" -ne 0 ] && [ "$API_KEY" = original ]`, "bash", home, name)
			wrapper.Env = direct.Env
			if output, err := wrapper.CombinedOutput(); err != nil {
				t.Fatalf("wrapper applied a failed result: %v %s", err, output)
			}
		}
	})
	t.Run("mise", func(t *testing.T) {
		t.Chdir(t.TempDir())
		backend, err := activation.Get("mise")
		if err != nil {
			t.Fatal(err)
		}
		if err := backend.Install(activation.Config{ProviderSlug: "bitwarden", FolderName: "Fixture", FolderID: "folder-1", ItemIDs: []string{"item-1"}}); err != nil {
			t.Fatal(err)
		}
		root, _ := os.Getwd()
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		bash, err := exec.LookPath("bash")
		if err != nil {
			t.Skip("bash not installed")
		}
		cmd := exec.Command(bash, "-c", `set -e; cd "$2"; source "$1/.bwenv.mise.sh"; source "$1/.bwenv.mise.sh"; [ "$API_KEY" = fake-secret-value ]; [ "${DB_URL+x}" != x ]`, "bash", root, base)
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "BW_SESSION=fixture-session", "XDG_CONFIG_HOME="+t.TempDir())
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("mise source from unrelated directory: %v\n%s", err, output)
		} else if len(output) != 0 {
			t.Fatalf("automatic mise loads must be silent: %s", output)
		}
		if _, err := os.Stat(filepath.Join(root, ".bwenv_vars")); err != nil {
			t.Fatalf("variable metadata must belong to the source project: %v", err)
		}
		if files, err := filepath.Glob(filepath.Join(os.Getenv("XDG_CACHE_HOME"), "bwenv", "mise", "*.env")); err != nil || len(files) != 0 {
			t.Fatalf("mise persisted secrets: %v %v", files, err)
		}
		locked := exec.Command(bash, "-c", `source "$1/.bwenv.mise.sh"; source "$1/.bwenv.mise.sh"`, "bash", root)
		locked.Env = append(cmd.Env, "BW_SESSION=")
		if output, err := locked.CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("locked mise startup must stay silent: %v %s", err, output)
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("SHELL", "/bin/bash")
		if _, _, err := runshell.InstallWrapper(); err != nil {
			t.Fatal(err)
		}
		notice := exec.Command(bash, "-c", `set -e
source "$2/.bashrc"
source "$1/.bwenv.mise.sh"
_bwenv_login_notice
source "$1/.bwenv.mise.sh"
_bwenv_login_notice
_bwenv_login_notice
unset _BWENV_LOGIN_REQUIRED
_bwenv_login_notice
source "$1/.bwenv.mise.sh"
_bwenv_login_notice
export BW_SESSION=fixture-session
source "$1/.bwenv.mise.sh"
_bwenv_login_notice
[ -z "${_BWENV_LOGIN_REQUIRED:-}" ]
[ "$API_KEY" = fake-secret-value ]`, "bash", root, home)
		notice.Env = locked.Env
		if output, err := notice.CombinedOutput(); err != nil || strings.Count(string(output), "run bwenv login") != 2 {
			t.Fatalf("expected one login hint per entry and none after unlocking: %v %s", err, output)
		}
		t.Run("disable and lock", func(t *testing.T) {
			log := filepath.Join(t.TempDir(), "calls.log")
			script := `set -e
source "$2/.bashrc"
cd "$1"
source .bwenv.mise.sh
[ "$API_KEY" = fake-secret-value ]
bwenv disallow
[ "${API_KEY+x}" != x ]
before=$(wc -l < "$BWENV_FAKE_LOG")
source .bwenv.mise.sh
source .bwenv.mise.sh
[ "${API_KEY+x}" != x ]
[ "${_BWENV_LOGIN_REQUIRED+x}" != x ]
[ "$(wc -l < "$BWENV_FAKE_LOG")" = "$before" ]
bwenv login
[ "$API_KEY" = fake-secret-value ]
bwenv lock
[ "${API_KEY+x}" != x ]
[ "${BW_SESSION+x}" != x ]
[ "$_BWENV_LOCKED" = 1 ]
before=$(wc -l < "$BWENV_FAKE_LOG")
source .bwenv.mise.sh
source .bwenv.mise.sh
[ "${API_KEY+x}" != x ]
[ "$_BWENV_LOGIN_REQUIRED" = "$1" ] || { printf 'hint path %s expected %s\n' "$_BWENV_LOGIN_REQUIRED" "$1"; exit 11; }
[ "$(wc -l < "$BWENV_FAKE_LOG")" = "$before" ]
bwenv login
[ "${_BWENV_LOCKED+x}" != x ]
[ "$API_KEY" = fake-secret-value ]`
			lifecycle := exec.Command(bash, "-c", script, "bash", root, home)
			lifecycle.Env = append(cmd.Env, "BWENV_FAKE_LOG="+log)
			if output, err := lifecycle.CombinedOutput(); err != nil {
				t.Fatalf("mise disable/lock lifecycle: %v\n%s", err, output)
			}
		})
		noBW := t.TempDir()
		if err := os.Symlink(filepath.Join(bin, "bwenv"), filepath.Join(noBW, "bwenv")); err != nil {
			t.Fatal(err)
		}
		missing := exec.Command(bash, "-c", `source "$1/.bwenv.mise.sh"`, "bash", root)
		missing.Env = append(locked.Env, "PATH="+noBW+":/usr/bin:/bin")
		if output, err := missing.CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("mise startup without bw must stay silent: %v %s", err, output)
		}
		explicit := exec.Command(filepath.Join(bin, "bwenv"), "export", "--project", root)
		explicit.Env = locked.Env
		if output, err := explicit.CombinedOutput(); err == nil || !strings.Contains(string(output), "run bwenv login") {
			t.Fatalf("explicit export must explain a locked session: %v %s", err, output)
		}
		mise, err := exec.LookPath("mise")
		if err == nil {
			cmd = exec.Command(mise, "env", "--json")
			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "BW_SESSION=fixture-session", "MISE_TRUSTED_CONFIG_PATHS="+root, "MISE_DATA_DIR="+t.TempDir(), "MISE_CACHE_DIR="+t.TempDir(), "MISE_CONFIG_DIR="+t.TempDir(), "MISE_STATE_DIR="+t.TempDir(), "MISE_EXPERIMENTAL=1")
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("mise env: %v", err)
			}
			var env map[string]string
			if err := json.Unmarshal(output, &env); err != nil || env["API_KEY"] != "fake-secret-value" {
				t.Fatalf("mise did not load selected secret: %v", err)
			}
			if err := backend.Unapprove(); err != nil {
				t.Fatal(err)
			}
			disabled := exec.Command(mise, "env", "--json")
			disabled.Env = cmd.Env
			output, err = disabled.Output()
			env = nil
			if err != nil || json.Unmarshal(output, &env) != nil || env["API_KEY"] != "" || env["_BWENV_LOGIN_REQUIRED"] != "" {
				t.Fatalf("disabled mise project must stay silent and unloaded: %v", err)
			}
			if err := backend.Approve(); err != nil {
				t.Fatal(err)
			}
			locked := exec.Command(mise, "env", "--json")
			locked.Env = append(cmd.Env, "_BWENV_LOCKED=1")
			output, err = locked.Output()
			env = nil
			if err != nil || json.Unmarshal(output, &env) != nil || env["API_KEY"] != "" || env["_BWENV_LOGIN_REQUIRED"] != root {
				t.Fatalf("locked mise project must signal login without exporting: %v, hint=%q expected=%q, hasAPI=%v", err, env["_BWENV_LOGIN_REQUIRED"], root, env["API_KEY"] != "")
			}
		}
		if mise != "" {
			t.Run("source with Node shim", func(t *testing.T) {
				node := miseNode
				if node == "" {
					t.Skip("no installed mise Node runtime")
				}
				versionOutput, err := exec.Command(node, "--version").Output()
				if err != nil {
					t.Skip("Node runtime unavailable")
				}
				version := strings.TrimPrefix(strings.TrimSpace(string(versionOutput)), "v")
				content, err := os.ReadFile("mise.toml")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile("mise.toml", append(content, []byte("\n[tools]\nnode = \""+version+"\"\n")...), 0644); err != nil {
					t.Fatal(err)
				}
				dataDir := t.TempDir()
				installDir := filepath.Join(dataDir, "installs", "node")
				if err := os.MkdirAll(installDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Dir(filepath.Dir(node)), filepath.Join(installDir, version)); err != nil {
					t.Fatal(err)
				}
				shimDir := t.TempDir()
				if err := os.Symlink(mise, filepath.Join(shimDir, "node")); err != nil {
					t.Fatal(err)
				}
				jsDir := t.TempDir()
				js := "#!/usr/bin/env node\nprocess.stdout.write('[{\"id\":\"item-1\",\"name\":\"Test\",\"fields\":[{\"name\":\"API_KEY\",\"value\":\"fake-secret-value\"}]}]');\n"
				if err := os.WriteFile(filepath.Join(jsDir, "bw"), []byte(js), 0755); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, mise, "env", "--json")
				cmd.WaitDelay = time.Second
				var diagnostic bytes.Buffer
				cmd.Stderr = &diagnostic
				cmd.Env = append(os.Environ(), "PATH="+shimDir+":"+jsDir+":"+bin+":"+os.Getenv("PATH"), "BW_SESSION=fixture-session", "MISE_TRUSTED_CONFIG_PATHS="+root, "MISE_CACHE_DIR="+t.TempDir(), "MISE_CONFIG_DIR="+t.TempDir(), "MISE_STATE_DIR="+t.TempDir(), "MISE_EXPERIMENTAL=1", "MISE_AUTO_INSTALL=false", "MISE_SHIMS_DIR="+shimDir, "MISE_DATA_DIR="+dataDir)
				output, err := cmd.Output()
				if err != nil {
					t.Fatalf("mise source recursed through Node shim: %v", err)
				}
				var env map[string]string
				if err := json.Unmarshal(output, &env); err != nil || env["API_KEY"] != "fake-secret-value" {
					t.Fatalf("Node CLI did not export its secret: %v; %s", err, diagnostic.String())
				}
				locked := exec.CommandContext(ctx, mise, "env", "--json")
				locked.WaitDelay = time.Second
				locked.Env = append(cmd.Env, "BW_SESSION=")
				output, err = locked.Output()
				if err != nil {
					t.Fatalf("locked vault prevented mise tool activation: %v", err)
				}
				env = nil
				if err := json.Unmarshal(output, &env); err != nil {
					t.Fatal(err)
				}
				if _, exported := env["API_KEY"]; exported {
					t.Fatal("locked vault exported secrets")
				}
				if !strings.Contains(env["PATH"], filepath.Join(installDir, version, "bin")) {
					t.Fatal("locked vault prevented Node activation for login")
				}
			})
		}
		if err := backend.Remove(); err != nil {
			t.Fatal(err)
		}
		if remaining, err := os.ReadFile("mise.toml"); err == nil && strings.Contains(string(remaining), "_.source") {
			t.Fatal("generated mise source left behind")
		}
	})
}

// Both installation paths must preserve release metadata and eval-safe failures.
func TestEntrypointCompatibility(t *testing.T) {
	for _, source := range []string{".", "./cmd/bwenv"} {
		t.Run(source, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "bwenv")
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			if output, err := exec.Command("go", "build", "-ldflags", "-X main.Version=v3.0.0-test", "-o", binary, source).CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}} {
				output, err := exec.Command(binary, args...).Output()
				if err != nil || !bytes.Contains(output, []byte("v3.0.0-test")) {
					t.Fatalf("version %v: %v\n%s", args, err, output)
				}
			}
			for _, args := range [][]string{{"export", "--unknown"}, {"export", "--provider"}, {"typo"}, {"hook", "invalid-shell"}} {
				command := exec.Command(binary, args...)
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				if err := command.Run(); err == nil || stdout.Len() != 0 || stderr.Len() == 0 {
					t.Fatalf("unsafe failure %v: err=%v stdout=%q stderr=%q", args, err, stdout.String(), stderr.String())
				}
			}
		})
	}
}
