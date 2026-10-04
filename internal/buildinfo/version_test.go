package buildinfo

import "testing"

func TestResolve(t *testing.T) {
	if got := Resolve("v3.0.0-rc.1"); got != "v3.0.0-rc.1" {
		t.Fatalf("injected version changed: %q", got)
	}
	if got := Resolve(""); got == "" {
		t.Fatal("plain builds need version metadata")
	}
}
