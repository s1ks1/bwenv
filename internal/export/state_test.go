package export

import (
	"encoding/base64"
	"os"
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
	for _, data := range []string{`null`, `{}`, `{"Root":"/tmp","Values":{"BAD;touch injected":null}}`} {
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
