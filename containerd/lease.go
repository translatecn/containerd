package containerd

import (
	"context"
	leases2 "demo/over/leases"
	"time"
)

// WithLease attaches a lease on the context
func (c *Client) WithLease(ctx context.Context, opts ...leases2.Opt) (context.Context, func(context.Context) error, error) {
	nop := func(context.Context) error { return nil }

	_, ok := leases2.FromContext(ctx)
	if ok {
		return ctx, nop, nil
	}

	ls := c.LeasesService()

	if len(opts) == 0 {
		// Use default lease configuration if no options provided
		opts = []leases2.Opt{
			leases2.WithRandomID(),
			leases2.WithExpiration(24 * time.Hour),
		}
	}

	l, err := ls.Create(ctx, opts...)
	if err != nil {
		return ctx, nop, err
	}

	ctx = leases2.WithLease(ctx, l.ID)
	return ctx, func(ctx context.Context) error {
		return ls.Delete(ctx, l)
	}, nil
}
