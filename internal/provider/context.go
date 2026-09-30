package provider

import (
	"context"
	"time"
)

// DefaultTimeout bounds a provider command when the caller supplies a context
// without a deadline. Callers that set their own deadline or cancel the
// context keep full control.
const (
	DefaultTimeout     = 5 * time.Second
	DefaultSyncTimeout = 30 * time.Second
)

func WithDefaultTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
