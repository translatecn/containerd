package sanbox

import (
	metadata2 "demo/pkg/metadata"
	"demo/pkg/plugin"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.SandboxStorePlugin,
		ID:   "local",
		Requires: []plugin.Type{
			plugin.MetadataPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}

			return metadata2.NewSandboxStore(m.(*metadata2.DB)), nil
		},
	})
}
