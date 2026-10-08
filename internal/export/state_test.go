package export

import (
	"encoding/base64"
	"github.com/s1ks1/bwenv/v3/internal/activation"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/provider"
)

func TestEnvironmentStatePreservesOriginalAndRejectsTampering(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("API_KEY", "original ' \\ value\nnext line")
	encoded, err := rememberState([]provider.Secret{{Key: "API_KEY"}, {Key: "UNSET_TEST_KEY"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(stateVariable, encoded)
	t.Setenv("API_KEY", "project value")
	again, err := rememberState([]provider.Secret{{Key: "API_KEY"}})
	if err != nil || again != encoded {
		t.Fatalf("re-activation overwrote original state: %v", err)
	}
	state, err := decodeState(encoded)
	if err != nil || *state.Values["API_KEY"] != "original ' \\ value\nnext line" || state.Values["UNSET_TEST_KEY"] != nil {
		t.Fatalf("incorrect state: %v", err)
	}
	for _, data := range []string{`null`, `{}`, `{"Root":"/tmp","Values":{"API_KEY":"fixture\u0000bad"}}`, `{"Root":"/tmp","Values":{"BAD;touch injected":null}}`} {
		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		stdout, _, err := stabilityCaptureOutput(t, func() error { _, err := restoreState(encoded, "bash"); return err })
		if err == nil || stdout != "" {
			t.Fatal("tampered state emitted executable output")
		}
	}
	if err := os.WriteFile(".bwenv_vars", []byte("API_KEY\nBAD;touch injected\n$(touch injected)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	names := loadCachedVarNames()
	if len(names) != 1 || names[0] != "API_KEY" {
		t.Fatalf("unsafe cached names: %v", names)
	}
}

func TestDiskNamesCannotExecuteThroughEval(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("requires bash")
	}
	t.Chdir(t.TempDir())
	activation.Register(&stubBackend{})
	if err := os.WriteFile(".bwenv.toml", []byte(stubProjectConfig), 0600); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(t.TempDir(), "injected")
	for _, filename := range []string{".bwenv_vars", ".envrc"} {
		content := "GOOD\nBW_SESSION\nBAD;touch " + sentinel + "\n$(touch " + sentinel + ")\n"
		if filename == ".envrc" {
			os.Remove(".bwenv_vars")
			content = "export GOOD='value'\nexport BW_SESSION='session'\nexport BAD;touch " + sentinel + "=value\nexport $(touch " + sentinel + ")=value\n"
		}
		if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		stdout, _, err := stabilityCaptureOutput(t, func() error { _, err := DisallowAndUnset(); return err })
		if err != nil || strings.Contains(stdout, "touch") {
			t.Fatalf("unsafe unset output from %s: %q %v", filename, stdout, err)
		}
		script := "export GOOD=value BW_SESSION=fixture\n" + stdout + "[ -z \"${GOOD+x}\" ] && [ -z \"${BW_SESSION+x}\" ] && [ ! -e \"$1\" ]"
		if output, err := exec.Command(bash, "-c", script, "bash", sentinel).CombinedOutput(); err != nil {
			t.Fatalf("eval validation: %v %s", err, output)
		}
	}
}
