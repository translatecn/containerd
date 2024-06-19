package content

import (
	"demo/over/plugin"
	"demo/plugins"
	contentserver "demo/plugins/containerd/content/over_contentserver"
	"errors"

	"demo/over/content"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "content",
		Requires: []plugin.Type{
			plugin.ServicePlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			_plugins, err := ic.GetByType(plugin.ServicePlugin)
			if err != nil {
				return nil, err
			}
			p, ok := _plugins[plugins.ContentService]
			if !ok {
				return nil, errors.New("content store service not found")
			}
			cs, err := p.Instance()
			if err != nil {
				return nil, err
			}
			return contentserver.New(cs.(content.Store)), nil
		},
	})
}
