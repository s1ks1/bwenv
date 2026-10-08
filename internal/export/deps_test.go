package export

import (
	"os/exec"
	"strings"
	"testing"
)

// Guard: the export package must depend only on the activation interface, never
// on a concrete backend, so secret retrieval stays backend-agnostic. If a future
// change imports activation/direnv into export, this test fails.
func TestExportDoesNotDependOnConcreteActivationBackend(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v", err)
	}

	for _, dep := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasSuffix(dep, "internal/activation/direnv") {
			t.Fatalf("export must not import a concrete activation backend, found %q", dep)
		}
	}
}
