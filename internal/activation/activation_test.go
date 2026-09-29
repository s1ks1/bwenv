package activation

import (
	"strings"
	"testing"
)

type stubActivator struct{ name string }

func (s stubActivator) Name() string                  { return s.name }
func (s stubActivator) Available() bool               { return true }
func (s stubActivator) Detect() Status                { return Status{} }
func (s stubActivator) Render(Config) ([]byte, error) { return nil, nil }
func (s stubActivator) Install(Config) error          { return nil }
func (s stubActivator) Remove() error                 { return nil }
func (s stubActivator) Resolve() (Source, error)      { return Source{}, nil }
func (s stubActivator) Approve() error                { return nil }
func (s stubActivator) Unapprove() error              { return nil }
func (s stubActivator) Reload() error                 { return nil }

func TestGetIsCaseInsensitive(t *testing.T) {
	Register(stubActivator{name: "Stub"})

	a, err := Get("stub")
	if err != nil {
		t.Fatalf("Get(\"stub\") error: %v", err)
	}
	if a.Name() != "Stub" {
		t.Fatalf("Get returned %q, want Stub", a.Name())
	}
}

func TestGetUnknownModeListsAvailable(t *testing.T) {
	Register(stubActivator{name: "stub"})

	_, err := Get("does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown activation mode")
	}
	if !strings.Contains(err.Error(), "stub") {
		t.Fatalf("error should list available modes, got: %v", err)
	}
}
