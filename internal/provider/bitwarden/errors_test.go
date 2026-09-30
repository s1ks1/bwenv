package bitwarden

import (
	"context"
	"errors"
	"testing"

	"github.com/s1ks1/bwenv/v3/internal/process"
	"github.com/s1ks1/bwenv/v3/internal/provider"
)

// outputRunner returns canned output for every provider CLI invocation.
type outputRunner struct {
	stdout []byte
	err    error
}

func (r outputRunner) Run(context.Context, string, []string, process.IO) (process.Result, error) {
	return process.Result{Stdout: r.stdout}, r.err
}

func TestListFoldersTypedErrors(t *testing.T) {
	cases := []struct {
		name   string
		runner outputRunner
		want   error
	}{
		{"empty output is an expired session", outputRunner{}, provider.ErrSessionExpired},
		{"non-json output is malformed", outputRunner{stdout: []byte("Your vault is locked.")}, provider.ErrMalformedProviderResponse},
		{"command failure is unavailable", outputRunner{err: errors.New("boom")}, provider.ErrProviderUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (&Bitwarden{Runner: tc.runner}).ListFolders(context.Background(), "session")
			if !errors.Is(err, tc.want) {
				t.Fatalf("ListFolders() error = %v, want errors.Is %v", err, tc.want)
			}
		})
	}
}

func TestMissingSelectedItemIsTyped(t *testing.T) {
	runner := outputRunner{stdout: []byte(`[{"id":"item-1","name":"A"}]`)}
	_, err := (&Bitwarden{Runner: runner}).GetSecretsByItemIDs(context.Background(), "s", provider.Folder{ID: "f", Name: "F"}, []string{"missing"})
	if !errors.Is(err, provider.ErrItemNotFound) {
		t.Fatalf("error = %v, want ErrItemNotFound", err)
	}
}
