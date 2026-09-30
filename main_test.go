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
	bin := t.TempDir()
	for target, source := range map[string]string{"bwenv": ".", "bw": "./tests/fixtures/fakecli"} {
		if output, err := exec.Command("go", "build", "-o", filepath.Join(bin, target), source).CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", target, err, output)
		}
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
set -q _BWENV_STATE; and exit 8`
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
	t.Run("plain login", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
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
cd "$1"
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
		locked := exec.Command(bash, "-c", `source "$1/.bwenv.mise.sh"; source "$1/.bwenv.mise.sh"`, "bash", root)
		locked.Env = append(cmd.Env, "BW_SESSION=")
		if output, err := locked.CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("locked mise startup must stay silent: %v %s", err, output)
		}
		home := t.TempDir()
		t.Setenv("HOME", home)
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
		}
		if mise != "" {
			t.Run("source with Node shim", func(t *testing.T) {
				lookup := exec.Command(mise, "which", "node")
				lookup.Dir = base
				nodeOutput, err := lookup.Output()
				if err != nil {
					t.Skip("no installed mise Node runtime")
				}
				node := strings.TrimSpace(string(nodeOutput))
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
				cmd.Env = append(os.Environ(), "PATH="+shimDir+":"+jsDir+":"+bin+":"+os.Getenv("PATH"), "BW_SESSION=fixture-session", "MISE_TRUSTED_CONFIG_PATHS="+root, "MISE_CACHE_DIR="+t.TempDir(), "MISE_CONFIG_DIR="+t.TempDir(), "MISE_STATE_DIR="+t.TempDir(), "MISE_EXPERIMENTAL=1", "MISE_AUTO_INSTALL=false", "MISE_SHIMS_DIR="+shimDir)
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
				if !strings.Contains(env["PATH"], filepath.Dir(node)) {
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
