package cgroups

import (
	"demo/others/cgroups/v3"
	"demo/over/events"
	v1 "demo/over/metrics/cgroups/v1"
	v2 "demo/over/metrics/cgroups/v2"
	"demo/over/platforms"
	"demo/over/plugin"
	"demo/over/runtime"
	metrics "github.com/docker/go-metrics"
)

// Config for the cgroups monitor
type Config struct {
	NoPrometheus bool `toml:"no_prometheus"`
}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.TaskMonitorPlugin,
		ID:     "cgroups",
		InitFn: New,
		Requires: []plugin.Type{
			plugin.EventPlugin,
		},
		Config: &Config{},
	})
}

// New returns a new cgroups monitor
func New(ic *plugin.InitContext) (interface{}, error) {
	var ns *metrics.Namespace
	config := ic.Config.(*Config)
	if !config.NoPrometheus {
		ns = metrics.NewNamespace("container", "", nil)
	}
	var (
		tm  runtime.TaskMonitor
		err error
	)

	ep, err := ic.Get(plugin.EventPlugin)
	if err != nil {
		return nil, err
	}

	if cgroups.Mode() == cgroups.Unified {
		tm, err = v2.NewTaskMonitor(ic.Context, ep.(events.Publisher), ns)
	} else {
		tm, err = v1.NewTaskMonitor(ic.Context, ep.(events.Publisher), ns)
	}
	if err != nil {
		return nil, err
	}
	if ns != nil {
		metrics.Register(ns)
	}
	ic.Meta.Platforms = append(ic.Meta.Platforms, platforms.DefaultSpec())
	return tm, nil
}
