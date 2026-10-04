package export

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMalformedStateStillClearsSafeCachedNamesAndSessions(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable")
	}
	t.Chdir(t.TempDir())
	t.Setenv("PATH", t.TempDir()) // Never contact a real vault in this test.
	t.Setenv(stateVariable, "malformed")
	t.Setenv("BW_SESSION", "fixture")
	t.Setenv("OP_SESSION_unsafe;touch injected", "fixture")
	if err := os.WriteFile(bwenvVarsCacheFile, []byte("API_KEY\nBAD;touch injected\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := stabilityCaptureOutput(t, func() error {
		_, err := LockAndUnset(context.Background(), "bash")
		return err
	})
	if err == nil || strings.Contains(stdout, "touch") {
		t.Fatalf("unsafe cleanup or swallowed state error: %q %v", stdout, err)
	}
	cmd := exec.Command(bash, "-c", "export API_KEY=fixture BW_SESSION=fixture\n"+stdout+`[ "${API_KEY+x}" != x ] && [ "${BW_SESSION+x}" != x ] && [ "${_BWENV_STATE+x}" != x ] && [ "$_BWENV_LOCKED" = 1 ] && [ ! -e injected ]`)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("safe cleanup did not apply: %v %s", err, output)
	}
}

func TestExportCannotOverwriteHookControlVariables(t *testing.T) {
	for _, key := range []string{"_BWENV_LOCKED", "_bwenv_active_root"} {
		t.Run(key, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("BW_SESSION", "fixture")
			t.Setenv(lockedVariable, "")
			stabilityInstallFakeBW(t, `printf '%s' '[{"id":"item-1","fields":[{"name":"`+key+`","value":"fixture"}]}]'`)
			stdout, _, err := stabilityCaptureOutput(t, func() error {
				return ExportQuiet(context.Background(), "bitwarden", "Fixture", "folder-1", []string{"item-1"})
			})
			if err == nil || !strings.Contains(err.Error(), "reserved") || stdout != "" {
				t.Fatalf("reserved name was emitted: %q %v", stdout, err)
			}
			stdout, stderr, err := stabilityCaptureOutput(t, func() error {
				return ExportWithFolderID(context.Background(), "bitwarden", "Fixture", "folder-1", []string{"item-1"})
			})
			if err == nil || stdout != "" || !strings.Contains(stderr, "Invalid variable") {
				t.Fatalf("explicit export did not explain rejection: %q %q %v", stdout, stderr, err)
			}
			if _, err := os.Stat(bwenvVarsCacheFile); !os.IsNotExist(err) {
				t.Fatal("invalid export wrote variable metadata")
			}
		})
	}
}

func TestNULValuesAreRejectedBeforeAnyShellOutput(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("BW_SESSION", "fixture")
	t.Setenv(lockedVariable, "")
	stabilityInstallFakeBW(t, `printf '%s' '[{"id":"item-1","fields":[{"name":"SAFE","value":"good"},{"name":"API_KEY","value":"fixture-secret\u0000bad"}]}]'`)
	stdout, stderr, err := stabilityCaptureOutput(t, func() error {
		return ExportWithFolderID(context.Background(), "bitwarden", "Fixture", "folder-1", []string{"item-1"})
	})
	if err == nil || stdout != "" || strings.Contains(stderr, "fixture-secret") {
		t.Fatalf("unsafe NUL failure: stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	if _, err := os.Stat(bwenvVarsCacheFile); !os.IsNotExist(err) {
		t.Fatal("invalid export wrote variable metadata")
	}
}
