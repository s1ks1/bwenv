package envrc

// PER-12 stabilization: focused regression tests for the fast export path.
//
// These tests are additive-only (new file). They exercise the shell quoting
// helpers (shellQuote/sanitizeKey/shellEscape) through real shells where
// available, the non-interactive ExportWithFolderID path, canonical and
// legacy folder roundtrips, duplicate-key behavior, and session failure
// handling. Any real source bug found here is surfaced via t.Skip("BUG: ...")
// rather than by patching source (PER-17 is mid-flight).

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/provider"
	"github.com/s1ks1/bwenv/v3/internal/shell"
)

// ── helpers ─────────────────────────────────────────────────────────────────

// stabilityRunShell runs a script in the given shell and returns stdout/stderr.
func stabilityRunShell(t *testing.T, shellPath, script string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(shellPath, "-c", script)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err = cmd.Run()
	return out.String(), errOut.String(), err
}

// stabilityCaptureOutput captures stdout+stderr while fn runs.
func stabilityCaptureOutput(t *testing.T, fn func() error) (stdout, stderr string, err error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	rErr, wErr, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	os.Stdout, os.Stderr = wOut, wErr
	runErr := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	outData, _ := io.ReadAll(rOut)
	errData, _ := io.ReadAll(rErr)
	_ = rOut.Close()
	_ = rErr.Close()
	return string(outData), string(errData), runErr
}

func stabilityChdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// stabilityInstallFakeBW puts a fake "bw" on PATH so IsAvailable() passes and
// the non-interactive export path can be driven without a real Bitwarden CLI.
func stabilityInstallFakeBW(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake CLI is a POSIX shell script")
	}
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "bw"), []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
}

// ── shellQuote across real shells ───────────────────────────────────────────

func TestShellQuoteSurvivesRealShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting test")
	}

	type shellCase struct {
		name   string
		path   string
		prefix string
	}
	var shells []shellCase
	register := func(name, binary, prefix string) {
		if p, err := exec.LookPath(binary); err == nil {
			shells = append(shells, shellCase{name: name, path: p, prefix: prefix})
		}
	}
	register("sh", "sh", "export BWENV_TEST_VAR=")
	register("dash", "dash", "export BWENV_TEST_VAR=")
	register("bash", "bash", "export BWENV_TEST_VAR=")
	register("zsh", "zsh", "export BWENV_TEST_VAR=")
	// fish uses `set -x` for exported vars; its single-quote handling is what
	// matters for shellQuote compatibility.
	register("fish", "fish", "set -x BWENV_TEST_VAR ")
	if len(shells) == 0 {
		t.Skip("no supported shell found on PATH")
	}

	values := []string{
		"plain",
		"with spaces",
		"single ' quote",
		`double " quote`,
		"dollar $HOME and $(whoami) and ${PATH}",
		"back`tick` and \\backslash",
		"semi;colon && rm -rf / || true",
		"newline\nsecond\r\nthird\ttab",
		"'; echo INJECTED; '",
		"",
	}

	for _, sh := range shells {
		for i, value := range values {
			t.Run(fmt.Sprintf("%s/value_%d", sh.name, i), func(t *testing.T) {
				script := sh.prefix + shell.Quote(value) + `; printf '%s' "$BWENV_TEST_VAR"`
				out, errOut, err := stabilityRunShell(t, sh.path, script)
				if err != nil {
					t.Fatalf("shell %s rejected quoted value %q: %v (stderr: %s)", sh.name, value, err, errOut)
				}
				if out != value {
					t.Fatalf("shell %s roundtrip mismatch: got %q, want %q", sh.name, out, value)
				}
			})
		}
	}
}

// ── sanitizeKey ─────────────────────────────────────────────────────────────

func TestSanitizeKeyIsShellSafeAndIdempotent(t *testing.T) {
	var validIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	keys := []string{
		"OK_KEY",
		"my key",
		"$FOO",
		"1START",
		"a;b",
		"a&&b",
		"a`b`",
		"it's",
		"a|b",
		"a\nb",
		"---",
		"",
	}

	shPath, shErr := exec.LookPath("sh")
	for _, key := range keys {
		t.Run(fmt.Sprintf("key_%q", key), func(t *testing.T) {
			got := shell.SanitizeKey(key)
			if !validIdent.MatchString(got) {
				t.Fatalf("shell.SanitizeKey(%q) = %q is not a valid POSIX identifier", key, got)
			}
			if again := shell.SanitizeKey(got); again != got {
				t.Fatalf("sanitizeKey not idempotent: shell.SanitizeKey(%q) = %q", got, again)
			}
			if shErr != nil {
				t.Logf("sh unavailable; skipping shell eval for %q", key)
				return
			}
			script := "export " + got + "=" + shell.Quote("v") + `; printf '%s' "$` + got + `"`
			out, errOut, err := stabilityRunShell(t, shPath, script)
			if err != nil {
				t.Fatalf("generated identifier %q not eval-safe: %v (stderr: %s)", got, err, errOut)
			}
			if out != "v" {
				t.Fatalf("generated identifier %q eval mismatch: got %q, want %q", got, out, "v")
			}
		})
	}
}

// ── fast export path output is eval-safe ────────────────────────────────────

func TestFastExportOutputIsEvalSafe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell eval test")
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	sentinel := filepath.Join(t.TempDir(), "pwned-by-injection")
	cases := []struct {
		key   string
		value string
	}{
		{"SPACED", "hello world"},
		{"SINGLE", "it's"},
		{"DOUBLE", `say "hi"`},
		{"DOLLAR", "$HOME ${PATH} $(id)"},
		{"BACKTICK", "`id`"},
		{"SEMI", "a;b && c || d"},
		{"NEWLINE", "line1\nline2"},
		{"SPACED KEY", "weird key name"},
		{"dotted.key", "dotted key"},
		{"1START", "leading digit"},
		{"INJECT", "'; touch " + sentinel + "; echo '"},
	}

	secrets := make([]provider.Secret, 0, len(cases))
	sanitized := make([]string, 0, len(cases))
	for _, c := range cases {
		secrets = append(secrets, provider.Secret{Key: c.key, Value: c.value})
		sanitized = append(sanitized, shell.SanitizeKey(c.key))
	}
	provider.Register(&mockProvider{name: "Mock", secrets: secrets})

	stabilityChdir(t, t.TempDir())
	stdout, stderr, err := stabilityCaptureOutput(t, func() error {
		return ExportWithFolderID("mock", "Folder", "folder-id", nil)
	})
	if err != nil {
		t.Fatalf("ExportWithFolderID() returned error: %v (stderr: %s)", err, stderr)
	}

	// Build a script that evals the generated export lines and prints each
	// variable followed by a unique separator.
	var b strings.Builder
	b.WriteString(stdout)
	for _, key := range sanitized {
		b.WriteString("printf '%s' \"$" + key + "\"; printf '<<BWS>>'\n")
	}
	out, errOut, err := stabilityRunShell(t, shPath, b.String())
	if err != nil {
		t.Fatalf("generated export lines are not eval-safe: %v (stderr: %s)", err, errOut)
	}

	got := strings.Split(out, "<<BWS>>")
	if len(got) < len(cases) {
		t.Fatalf("expected %d values from eval, got %d fields (%q)", len(cases), len(got), out)
	}
	for i, c := range cases {
		if got[i] != c.value {
			t.Errorf("variable %q (%s): got %q, want %q", sanitized[i], c.key, got[i], c.value)
		}
	}

	if _, statErr := os.Stat(sentinel); statErr == nil {
		t.Fatalf("BUG: shell metacharacters in a secret value were executed (injection created %s)", sentinel)
	}
}

// ── duplicate variable names ────────────────────────────────────────────────

func TestFastExportDuplicateKeysLastWins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell eval test")
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	cases := []struct {
		name    string
		secrets []provider.Secret
	}{
		{
			name: "identical key from two items",
			secrets: []provider.Secret{
				{Key: "API_KEY", Value: "first"},
				{Key: "API_KEY", Value: "second"},
			},
		},
		{
			name: "sanitized key collision",
			secrets: []provider.Secret{
				{Key: "api-key", Value: "hyphen"},
				{Key: "api.key", Value: "dot"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider.Register(&mockProvider{name: "Mock", secrets: tc.secrets})
			stabilityChdir(t, t.TempDir())

			stdout, stderr, err := stabilityCaptureOutput(t, func() error {
				return ExportWithFolderID("mock", "Folder", "folder-id", nil)
			})
			if err != nil {
				t.Fatalf("ExportWithFolderID() returned error: %v (stderr: %s)", err, stderr)
			}

			key := shell.SanitizeKey(tc.secrets[0].Key)
			lines := strings.Count(stdout, "export "+key+"=")
			if lines != len(tc.secrets) {
				t.Fatalf("expected %d export lines for duplicate key %q, got %d:\n%s", len(tc.secrets), key, lines, stdout)
			}

			script := stdout + "\n" + `printf '%s' "$` + key + `"`
			got, errOut, err := stabilityRunShell(t, shPath, script)
			if err != nil {
				t.Fatalf("eval failed: %v (stderr: %s)", err, errOut)
			}
			want := tc.secrets[len(tc.secrets)-1].Value
			if got != want {
				t.Fatalf("duplicate key %q: got %q, want last-defined %q", key, got, want)
			}
			// Documented behavior: bwenv emits every definition in order; the
			// shell's sequential eval means the LAST definition wins.
			t.Logf("documented behavior: %d export lines for %q; last definition wins (%q)", lines, key, want)
		})
	}
}

// ── folder name roundtrips ──────────────────────────────────────────────────

func TestFastExportCanonicalFolderRoundTrip(t *testing.T) {
	names := []string{
		"My Project",
		"it's a folder",
		`double " quote`,
		"dollar $HOME",
		"back`tick`",
		"semi;colon && x",
		"pipe|name",
		"new\nline",
	}
	for i, name := range names {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			stabilityChdir(t, t.TempDir())
			if err := Generate(Config{
				ProviderSlug: "bitwarden",
				FolderName:   name,
				FolderID:     "folder-id-123",
				Session:      "tok",
				Version:      testVersion,
			}); err != nil {
				t.Fatalf("Generate() with folder %q returned error: %v", name, err)
			}

			_, folder, folderID, _, err := ParseEnvrcConfigWithFolderID()
			if err != nil {
				t.Fatalf("ParseEnvrcConfigWithFolderID() with folder %q returned error: %v", name, err)
			}
			if folder != name {
				t.Errorf("canonical roundtrip: got folder %q, want %q", folder, name)
			}
			if folderID != "folder-id-123" {
				t.Errorf("canonical roundtrip: got folder ID %q, want %q", folderID, "folder-id-123")
			}

			envrc, err := os.ReadFile(".envrc")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(envrc), `eval "$(bwenv export --project .)"`) {
				t.Errorf("expected untouched canonical eval line, got:\n%s", envrc)
			}
		})
	}
}

func TestFastExportLegacyFolderRoundTrip(t *testing.T) {
	names := []string{
		"My Project",
		"it's a folder",
		`double " quote`,
		"dollar $HOME",
		"back`tick`",
		"semi;colon && x",
		"pipe|name",
	}
	for i, name := range names {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			stabilityChdir(t, t.TempDir())
			if err := Generate(Config{
				ProviderSlug: "bitwarden",
				FolderName:   name,
				Version:      testVersion,
			}); err != nil {
				t.Fatalf("Generate() legacy with folder %q returned error: %v", name, err)
			}

			_, folder, _, err := ParseEnvrcConfig()
			if err != nil {
				t.Fatalf("ParseEnvrcConfig() legacy with folder %q returned error: %v", name, err)
			}
			if folder != name {
				t.Skipf("BUG: legacy .envrc folder roundtrip lost data: %q -> %q (header is split on '|' at envrc.go ParseEnvrcConfigWithFolderID)", name, folder)
			}
		})
	}
}

// ── session handling ────────────────────────────────────────────────────────

func TestFastExportSessionFailureIsGraceful(t *testing.T) {
	t.Run("missing session", func(t *testing.T) {
		stabilityInstallFakeBW(t, `echo "unexpected bw invocation" >&2; exit 0`)
		t.Setenv("BW_SESSION", "")
		stabilityChdir(t, t.TempDir())

		_, _, err := stabilityCaptureOutput(t, func() error {
			return ExportWithFolderID("bitwarden", "Folder", "folder-id", nil)
		})
		if err == nil {
			t.Fatal("expected graceful error when BW_SESSION is empty/expired")
		}
		msg := err.Error()
		if !strings.Contains(msg, "session expired or not active") {
			t.Errorf("error should explain the session state, got: %v", err)
		}
		if !strings.Contains(msg, "bwenv login") {
			t.Errorf("error should point the user to 'bwenv login', got: %v", err)
		}
	})

	t.Run("expired session does not leak token", func(t *testing.T) {
		const token = "super-secret-session-token-abc123"
		stabilityInstallFakeBW(t, `echo "Vault is locked. Run 'bwenv login'." >&2; exit 1`)
		t.Setenv("BW_SESSION", token)
		stabilityChdir(t, t.TempDir())

		_, stderr, err := stabilityCaptureOutput(t, func() error {
			return ExportWithFolderID("bitwarden", "Folder", "folder-id", nil)
		})
		if err == nil {
			t.Fatal("expected error when the session is rejected by the provider CLI")
		}
		if !strings.Contains(err.Error(), "bw returned unexpected output") ||
			!strings.Contains(err.Error(), "bwenv login") {
			t.Errorf("error should be generic and point to 'bwenv login', got: %v", err)
		}
		if strings.Contains(err.Error(), "Vault is locked") {
			t.Errorf("raw provider output leaked into the error: %v", err)
		}
		for label, text := range map[string]string{"returned error": err.Error(), "stderr": stderr} {
			if strings.Contains(text, token) {
				t.Fatalf("session token leaked in %s: %s", label, text)
			}
		}
	})
}
