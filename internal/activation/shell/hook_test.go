package shell

import (
	"strings"
	"testing"
)

func TestHookForEachShell(t *testing.T) {
	for _, name := range SupportedShells {
		snippet, err := Hook(name)
		if err != nil {
			t.Fatalf("Hook(%q) error: %v", name, err)
		}
		for _, want := range []string{hookMarker, "bwenv root", "bwenv activate", "bwenv deactivate"} {
			if !strings.Contains(snippet, want) {
				t.Fatalf("Hook(%q) missing %q:\n%s", name, want, snippet)
			}
		}
	}
}

func TestHookRejectsUnsupportedShell(t *testing.T) {
	if _, err := Hook("tcsh"); err == nil {
		t.Fatal("expected an error for an unsupported shell")
	}
}

func TestDetectShell(t *testing.T) {
	cases := map[string]string{
		"/bin/zsh":      "zsh",
		"/usr/bin/bash": "bash",
		"/usr/bin/fish": "fish",
		"":              "zsh",
		"/bin/sh":       "zsh",
	}
	for in, want := range cases {
		if got := DetectShell(in); got != want {
			t.Errorf("DetectShell(%q) = %q, want %q", in, got, want)
		}
	}
}
