package snapshots

import (
	"demo/over/metadata"
	"demo/over/plugin"
	"demo/plugins"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.ServicePlugin,
		ID:   plugins.SnapshotsService,
		Requires: []plugin.Type{
			plugin.MetadataPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}

			return m.(*metadata.DB).Snapshotters(), nil
		},
	})
}
