package session

import (
	"strings"
	"testing"
)

func TestReauthenticateUnknownProvider(t *testing.T) {
	_, err := Reauthenticate("does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown provider")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("error should name the provider, got: %v", err)
	}
}
