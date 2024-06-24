package containers

import (
	"context"
	api "demo/over/api/services/containers/v1"
	"demo/over/plugin"
	"demo/over/plugins"
	ptypes "demo/over/protobuf/types"
	"errors"
	"google.golang.org/grpc"
	"io"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "containers",
		Requires: []plugin.Type{
			plugin.ServicePlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			_plugins, err := ic.GetByType(plugin.ServicePlugin)
			if err != nil {
				return nil, err
			}
			p, ok := _plugins[plugins.ContainersService]
			if !ok {
				return nil, errors.New("containers service not found")
			}
			i, err := p.Instance()
			if err != nil {
				return nil, err
			}
			return &service{local: i.(api.ContainersClient)}, nil
		},
	})
}

type service struct {
	local api.ContainersClient
	api.UnimplementedContainersServer
}

var _ api.ContainersServer = &service{}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterContainersServer(server, s)
	return nil
}

func (s *service) Get(ctx context.Context, req *api.GetContainerRequest) (*api.GetContainerResponse, error) {
	return s.local.Get(ctx, req)
}

func (s *service) List(ctx context.Context, req *api.ListContainersRequest) (*api.ListContainersResponse, error) {
	return s.local.List(ctx, req)
}

func (s *service) ListStream(req *api.ListContainersRequest, stream api.Containers_ListStreamServer) error {
	containers, err := s.local.ListStream(stream.Context(), req)
	if err != nil {
		return err
	}
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			c, err := containers.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if err := stream.Send(c); err != nil {
				return err
			}
		}
	}
}

func (s *service) Create(ctx context.Context, req *api.CreateContainerRequest) (*api.CreateContainerResponse, error) {
	return s.local.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, req *api.UpdateContainerRequest) (*api.UpdateContainerResponse, error) {
	return s.local.Update(ctx, req)
}

func (s *service) Delete(ctx context.Context, req *api.DeleteContainerRequest) (*ptypes.Empty, error) {
	return s.local.Delete(ctx, req)
}
