package task

import (
	"demo/pkg/plugin"
	"demo/pkg/plugins/shim/shim"
	"demo/pkg/runtime/v2/runc/task"
	"demo/pkg/shutdown"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.TTRPCPlugin,
		ID:   "task",
		Requires: []plugin.Type{
			plugin.EventPlugin,
			plugin.InternalPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			pp, err := ic.GetByID(plugin.EventPlugin, "publisher")
			if err != nil {
				return nil, err
			}
			ss, err := ic.GetByID(plugin.InternalPlugin, "shutdown")
			if err != nil {
				return nil, err
			}
			return task.NewTaskService(ic.Context, pp.(shim.Publisher), ss.(shutdown.Service))
		},
	})
}
