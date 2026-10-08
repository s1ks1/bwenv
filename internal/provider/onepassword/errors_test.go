package onepassword

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
	if _, err := (&OnePassword{Runner: outputRunner{err: errors.New("boom")}}).ListFolders(context.Background(), "s"); !errors.Is(err, provider.ErrProviderUnavailable) {
		t.Fatalf("command failure error = %v, want ErrProviderUnavailable", err)
	}
	if _, err := (&OnePassword{Runner: outputRunner{stdout: []byte("not json")}}).ListFolders(context.Background(), "s"); !errors.Is(err, provider.ErrMalformedProviderResponse) {
		t.Fatalf("malformed output error = %v, want ErrMalformedProviderResponse", err)
	}
}
