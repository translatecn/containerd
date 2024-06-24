package event

import (
	"demo/over/plugin"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.EventPlugin,
		ID:   "exchange",
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			// TODO: In 2.0, create exchange since ic.Events will be removed
			return ic.Events, nil
		},
	})
}
