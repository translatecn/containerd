package lease

import (
	"context"
	"demo/over/gc"
	"demo/over/leases"
	metadata2 "demo/over/metadata"
	"demo/over/plugin"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.LeasePlugin,
		ID:   "manager",
		Requires: []plugin.Type{
			plugin.MetadataPlugin,
			plugin.GCPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			g, err := ic.Get(plugin.GCPlugin)
			if err != nil {
				return nil, err
			}
			return &local{
				Manager: metadata2.NewLeaseManager(m.(*metadata2.DB)),
				gc:      g.(gcScheduler),
			}, nil
		},
	})
}

type gcScheduler interface {
	ScheduleAndWait(context.Context) (gc.Stats, error)
}

type local struct {
	leases.Manager
	gc gcScheduler
}

func (l *local) Delete(ctx context.Context, lease leases.Lease, opts ...leases.DeleteOpt) error {
	var do leases.DeleteOptions
	for _, opt := range opts {
		if err := opt(ctx, &do); err != nil {
			return err
		}
	}

	if err := l.Manager.Delete(ctx, lease); err != nil {
		return err
	}

	if do.Synchronous {
		if _, err := l.gc.ScheduleAndWait(ctx); err != nil {
			return err
		}
	}

	return nil

}
