package introspection

import (
	context "context"
	"demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	"demo/plugins"
	"errors"

	api "demo/over/api/services/introspection/v1"
	"google.golang.org/grpc"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type:     plugin.GRPCPlugin,
		ID:       "introspection",
		Requires: []plugin.Type{plugin.ServicePlugin},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			_plugins, err := ic.GetByType(plugin.ServicePlugin)
			if err != nil {
				return nil, err
			}
			p, ok := _plugins[plugins.IntrospectionService]
			if !ok {
				return nil, errors.New("introspection service not found")
			}

			i, err := p.Instance()
			if err != nil {
				return nil, err
			}

			localClient, ok := i.(*Local)
			if !ok {
				return nil, errors.New("could not create a local client for introspection service")
			}
			localClient.UpdateLocal(ic.Root)

			return &server{
				local: localClient,
			}, nil
		},
	})
}

type server struct {
	local api.IntrospectionClient
	api.UnimplementedIntrospectionServer
}

var _ = (api.IntrospectionServer)(&server{})

func (s *server) Register(server *grpc.Server) error {
	api.RegisterIntrospectionServer(server, s)
	return nil
}

func (s *server) Plugins(ctx context.Context, req *api.PluginsRequest) (*api.PluginsResponse, error) {
	return s.local.Plugins(ctx, req)
}

func (s *server) Server(ctx context.Context, empty *ptypes.Empty) (*api.ServerResponse, error) {
	return s.local.Server(ctx, empty) // ctr_bin plugin ls
}
