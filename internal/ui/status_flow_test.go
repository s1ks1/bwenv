package ui

import (
	"os"
	"testing"
)

func TestCheckEnvrcStatusReportsInvalidCanonicalProject(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.WriteFile(".bwenv.toml", []byte("version = 99\n"), 0600); err != nil {
		t.Fatal(err)
	}

	status := checkEnvrcStatus()
	if status.state != envrcInvalid || !status.canonical || status.problem == "" {
		t.Fatalf("status = %+v, want invalid canonical project with a validation error", status)
	}
}
