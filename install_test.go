package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerReleaseArchiveURLs(t *testing.T) {
	for _, shell := range []string{"sh", "pwsh"} {
		t.Run(shell, func(t *testing.T) {
			executable, err := exec.LookPath(shell)
			if err != nil || (shell == "sh" && runtime.GOOS == "windows") {
				t.Skip(shell + " unavailable")
			}
			filename, start := "install.sh", "    # Strip 'v' prefix"
			prelude := "VERSION=v3.0.0; PLATFORM=darwin-arm64; GITHUB_REPO=s1ks1/bwenv\n"
			invocation := "printf '%s' \"$DOWNLOAD_URL\""
			expected := "https://github.com/s1ks1/bwenv/releases/download/v3.0.0/bwenv-3.0.0-darwin-arm64.tar.gz"
			args := []string{"-c"}
			if shell == "pwsh" {
				filename, start = "install.ps1", "    # Construct download URL."
				prelude = "$Version='v3.0.0'; $arch='amd64'; $GitHubRepo='s1ks1/bwenv'\n"
				invocation = "[Console]::Write($downloadUrl)"
				expected = "https://github.com/s1ks1/bwenv/releases/download/v3.0.0/bwenv-3.0.0-windows-amd64.zip"
				args = []string{"-NoProfile", "-NonInteractive", "-Command"}
			}
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			text := string(content)
			first, last := strings.Index(text, start), strings.Index(text, "    # Create temp directory.")
			if first < 0 || last <= first {
				t.Fatal("installer URL construction missing")
			}
			result, err := exec.Command(executable, append(args, prelude+text[first:last]+invocation)...).CombinedOutput()
			if err != nil || string(result) != expected {
				t.Fatalf("URL=%q want=%q err=%v", result, expected, err)
			}
		})
	}
}

func TestInstallersRequireExactValidChecksum(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bwenv.zip")
	data := []byte("fixture archive")
	if err := os.WriteFile(archive, data, 0600); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	for _, shell := range []string{"sh", "pwsh"} {
		t.Run(shell, func(t *testing.T) {
			if shell == "sh" && runtime.GOOS == "windows" {
				t.Skip("POSIX installer")
			}
			executable, err := exec.LookPath(shell)
			if err != nil {
				t.Skip(shell + " unavailable")
			}
			filename, start, end := "install.sh", "verify_checksum() {", "# -- Main --"
			prelude := `info() { :; }; success() { :; }; error() { exit 1; }; fetch() { printf '%s\n' "$TEST_CHECKSUMS"; }` + "\n"
			invocation := `verify_checksum "$TEST_ARCHIVE" bwenv.zip v3.0.0` + "\n"
			if shell == "pwsh" {
				filename, start, end = "install.ps1", "function Test-Checksum {", "# -- Add to PATH --"
				prelude = `$ErrorActionPreference='Stop'; function Write-Info {}; function Write-OK {}; function Write-Err { param($Message) throw $Message }; function Invoke-RestMethod { return $env:TEST_CHECKSUMS }` + "\n"
				invocation = `Test-Checksum -FilePath $env:TEST_ARCHIVE -ArchiveName bwenv.zip -Ver v3.0.0` + "\n"
			}
			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			text := string(content)
			first, last := strings.Index(text, start), strings.Index(text, end)
			if first < 0 || last <= first {
				t.Fatal("installer verification function missing")
			}
			script := prelude + text[first:last] + invocation
			for _, test := range []struct {
				name, sums string
				valid      bool
			}{
				{"valid", digest + "  bwenv.zip", true},
				{"missing", "", false},
				{"suffix is not archive", digest + "  bwenv.zip.sbom.json", false},
				{"duplicate", digest + "  bwenv.zip\n" + digest + "  bwenv.zip", false},
				{"malformed hash", "invalid  bwenv.zip", false},
				{"tampered", strings.Repeat("0", 64) + "  bwenv.zip", false},
			} {
				t.Run(test.name, func(t *testing.T) {
					args := []string{"-c", script}
					if shell == "pwsh" {
						args = []string{"-NoProfile", "-NonInteractive", "-Command", script}
					}
					command := exec.Command(executable, args...)
					command.Env = append(os.Environ(), "TEST_ARCHIVE="+archive, "TEST_CHECKSUMS="+test.sums)
					result, err := command.CombinedOutput()
					if (err == nil) != test.valid {
						t.Fatalf("valid=%v err=%v output=%s", test.valid, err, result)
					}
				})
			}
		})
	}
}
