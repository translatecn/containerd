package tasks

import (
	"demo/pkg/log"
	"demo/pkg/plugin"
	"demo/pkg/runtime"
	"errors"
)

var tasksServiceRequires = []plugin.Type{
	plugin.EventPlugin,
	plugin.RuntimePlugin,
	plugin.RuntimePluginV2,
	plugin.MetadataPlugin,
	plugin.TaskMonitorPlugin,
	plugin.WarningPlugin,
}

func loadV1Runtimes(ic *plugin.InitContext) (map[string]runtime.PlatformRuntime, error) {
	rt, err := ic.GetByType(plugin.RuntimePlugin)
	if err != nil {
		return nil, err
	}

	runtimes := make(map[string]runtime.PlatformRuntime)
	for _, rr := range rt {
		ri, err := rr.Instance()
		if err != nil {
			log.G(ic.Context).WithError(err).Warn("could not load runtime instance due to initialization error")
			continue
		}
		r := ri.(runtime.PlatformRuntime)
		runtimes[r.ID()] = r
	}

	if len(runtimes) == 0 {
		return nil, errors.New("no runtimes available to create task service")
	}
	return runtimes, nil
}
