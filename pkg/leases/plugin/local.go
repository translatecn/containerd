/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package plugin

import (
	"context"
	over_plugin2 "demo/over/plugin"

	"demo/pkg/gc"
	"demo/pkg/leases"
	"demo/pkg/metadata"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.LeasePlugin,
		ID:   "manager",
		Requires: []over_plugin2.Type{
			over_plugin2.MetadataPlugin,
			over_plugin2.GCPlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			m, err := ic.Get(over_plugin2.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			g, err := ic.Get(over_plugin2.GCPlugin)
			if err != nil {
				return nil, err
			}
			return &local{
				Manager: metadata.NewLeaseManager(m.(*metadata.DB)),
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
