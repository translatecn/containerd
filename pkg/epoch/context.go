package epoch

import (
	"context"
	"time"
)

type (
	epochKey struct{}
)

// WithSourceDateEpoch associates the context with the epoch.
func WithSourceDateEpoch(ctx context.Context, tm *time.Time) context.Context {
	return context.WithValue(ctx, epochKey{}, tm)
}

// FromContext returns the epoch associated with the context.
// FromContext does not fall back to read the SOURCE_DATE_EPOCH env var.
func FromContext(ctx context.Context) *time.Time {
	v := ctx.Value(epochKey{})
	if v == nil {
		return nil
	}
	return v.(*time.Time)
}
