package export

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/s1ks1/bwenv/v3/internal/activation"
	"github.com/s1ks1/bwenv/v3/internal/session"
)

// LockAndUnset always emits validated local cleanup, even when a provider
// fails to lock. The lock/logout wrapper applies this output on failure too,
// then preserves the command's failing exit status.
func LockAndUnset(ctx context.Context, shellName string) ([]string, error) {
	if err := validateShell(shellName); err != nil {
		return nil, err
	}
	var names []string
	var cleanupErr error
	state, hasState := os.LookupEnv(stateVariable)
	if hasState {
		names, cleanupErr = restoreState(state, shellName)
	}
	if !hasState || cleanupErr != nil {
		rootErr := enterProjectRoot()
		cleanupErr = errors.Join(cleanupErr, rootErr)
		if rootErr == nil {
			names = loadCachedVarNames()
			if !hasState {
				if backend, err := activation.ForProject(); err == nil {
					if emitter, ok := backend.(activation.Emitter); ok && emitter.EmitsExports() {
						// An inactive native project has already restored originals.
						names = nil
					}
				}
			}
			for _, name := range names {
				fmt.Print(unassignment(name, shellName))
			}
		}
	}
	// Setting a runtime guard also covers providers without session tokens.
	fmt.Print(assignment(lockedVariable, "1", shellName))
	for _, name := range []string{stateVariable, "BW_SESSION", "OP_SESSION", "OP_SERVICE_ACCOUNT_TOKEN", "_BWENV_LOGIN_REQUIRED", "_bwenv_active_root", "_bwenv_attempted_root", "_bwenv_attempted_session"} {
		fmt.Print(unassignment(name, shellName))
	}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OP_SESSION_") && validName(name) {
			fmt.Print(unassignment(name, shellName))
		}
	}
	return names, errors.Join(cleanupErr, session.LockAll(ctx))
}
