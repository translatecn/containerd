package nri

import (
	"demo/pkg/nri"
	"demo/pkg/plugin"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.NRIApiPlugin,
		ID:     "nri",
		Config: nri.DefaultConfig(),
		InitFn: initFunc,
	})
}

func initFunc(ic *plugin.InitContext) (interface{}, error) {
	l, err := nri.New(ic.Config.(*nri.Config))
	return l, err
}
