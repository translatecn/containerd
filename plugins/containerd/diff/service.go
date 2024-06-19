package diff

import (
	"context"
	"demo/over/plugin"
	"demo/plugins"
	"errors"

	diffapi "demo/over/api/services/diff/v1"
	"google.golang.org/grpc"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "diff",
		Requires: []plugin.Type{
			plugin.ServicePlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			_plugins, err := ic.GetByType(plugin.ServicePlugin)
			if err != nil {
				return nil, err
			}
			p, ok := _plugins[plugins.DiffService]
			if !ok {
				return nil, errors.New("diff service not found")
			}
			i, err := p.Instance()
			if err != nil {
				return nil, err
			}
			return &service{local: i.(diffapi.DiffClient)}, nil
		},
	})
}

type service struct {
	local diffapi.DiffClient
	diffapi.UnimplementedDiffServer
}

var _ diffapi.DiffServer = &service{}

func (s *service) Register(gs *grpc.Server) error {
	diffapi.RegisterDiffServer(gs, s)
	return nil
}

func (s *service) Apply(ctx context.Context, er *diffapi.ApplyRequest) (*diffapi.ApplyResponse, error) {
	return s.local.Apply(ctx, er)
}

func (s *service) Diff(ctx context.Context, dr *diffapi.DiffRequest) (*diffapi.DiffResponse, error) {
	return s.local.Diff(ctx, dr)
}
