package direnv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestControlFailuresAreBoundedAndActionable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake direnv")
	}
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	fake := filepath.Join(dir, "direnv")
	for _, action := range []string{"allow", "deny", "reload"} {
		t.Run(action, func(t *testing.T) {
			if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := runControl(context.Background(), action); err == nil || !strings.Contains(err.Error(), "direnv "+action+" failed") {
				t.Fatalf("failure not surfaced: %v", err)
			}
			if err := os.WriteFile(fake, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0755); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			start := time.Now()
			err := runControl(ctx, action)
			if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "try again") {
				t.Fatalf("timeout not actionable: %v", err)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("stalled command was not bounded")
			}
		})
	}
}
