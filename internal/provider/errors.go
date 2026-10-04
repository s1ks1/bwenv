package provider

import (
	"context"
	"errors"
)

// Typed provider errors let the application layer map failures to actionable
// user messages without parsing human-readable strings. Providers wrap these
// sentinels, so callers use errors.Is instead of matching error text.
var (
	// ErrNotAuthenticated means the provider refused the credentials or the
	// user is not signed in.
	ErrNotAuthenticated = errors.New("provider is not authenticated")

	// ErrSessionExpired means a previously valid session is no longer usable.
	ErrSessionExpired = errors.New("provider session expired")

	// ErrProviderUnavailable means the provider CLI is missing or a command
	// failed before producing usable output.
	ErrProviderUnavailable = errors.New("provider is unavailable")

	// ErrFolderNotFound means the requested folder or vault does not exist.
	ErrFolderNotFound = errors.New("provider folder not found")

	// ErrItemNotFound means the requested secret item does not exist.
	ErrItemNotFound = errors.New("provider item not found")

	// ErrMalformedProviderResponse means the CLI output could not be parsed.
	ErrMalformedProviderResponse = errors.New("malformed provider response")

	// ErrProviderTimeout means a provider command exceeded its deadline.
	ErrProviderTimeout = errors.New("provider command timed out")
)

// SafeError retains the cause for errors.Is/As without exposing runner or JSON
// error text, which can contain a decrypted value or a session token.
func SafeError(err error) error {
	if err == nil {
		return nil
	}
	return safeError{cause: err}
}

type safeError struct{ cause error }

func (e safeError) Error() string {
	if errors.Is(e.cause, context.DeadlineExceeded) {
		return "provider operation timed out"
	}
	if errors.Is(e.cause, context.Canceled) {
		return "provider operation cancelled"
	}
	return "provider operation failed"
}
func (e safeError) Unwrap() error { return e.cause }
