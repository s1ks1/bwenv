package provider

import (
	"context"
	"time"
)

// defaultTimeout bounds a provider command when the caller supplies a context
// without a deadline. Callers that set their own deadline or cancel the
// context keep full control.
const (
	defaultTimeout     = 5 * time.Second
	defaultSyncTimeout = 30 * time.Second
)

func withDefaultTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
