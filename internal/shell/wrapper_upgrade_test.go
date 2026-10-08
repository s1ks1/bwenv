package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExistingWrapperUpgradeClearsOnFailedLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell wrapper")
	}
	for _, name := range []string{"bash", "zsh", "fish"} {
		t.Run(name, func(t *testing.T) {
			path, err := exec.LookPath(name)
			if err != nil {
				t.Skip(name + " unavailable")
			}
			home, bin := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("SHELL", path)
			rc, err := DetectRC()
			if err != nil {
				t.Fatal(err)
			}
			old := strings.Replace(wrapperBashZsh, lockWrapperPOSIX, "", 1)
			notice := loginNoticePOSIX
			if name == "fish" {
				notice = loginNoticeFish
				old = strings.Replace(wrapperFish, lockWrapperFish, "", 1)
				old = strings.ReplaceAll(old, `"$(command bwenv $argv)"`, "(command bwenv $argv | string collect)")
			}
			if err := os.MkdirAll(filepath.Dir(rc), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(rc, []byte("# keep custom settings\n"+old+notice), 0644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := InstallWrapper(); err != nil {
				t.Fatal(err)
			}
			if changed, _, err := InstallWrapper(); err != nil || changed {
				t.Fatalf("upgrade is not idempotent: %v", err)
			}
			content, _ := os.ReadFile(rc)
			if !strings.Contains(string(content), "# keep custom settings") {
				t.Fatal("upgrade discarded custom settings")
			}
			if err := os.WriteFile(filepath.Join(bin, "bwenv"), []byte("#!/bin/sh\nprintf 'unset API_KEY\\n'\nexit 9\n"), 0755); err != nil {
				t.Fatal(err)
			}
			script := `source "$1"; export API_KEY=secret; bwenv lock; code=$?; [ "$code" -eq 9 ] && [ "${API_KEY+x}" != x ]`
			if name == "fish" {
				if err := os.WriteFile(filepath.Join(bin, "bwenv"), []byte("#!/bin/sh\nprintf 'set -e API_KEY\\n'\nexit 9\n"), 0755); err != nil {
					t.Fatal(err)
				}
				script = `source "$argv[1]"; set -gx API_KEY secret; bwenv lock; set code $status; test $code -eq 9; and not set -q API_KEY`
			}
			cmd := exec.Command(path, "-c", script, name, rc)
			if name == "fish" {
				cmd = exec.Command(path, "-c", script, rc)
			}
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("failed lock must clean up and retain exit status: %v %s", err, output)
			}
		})
	}
}
