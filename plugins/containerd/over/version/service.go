package version

import (
	"context"
	"demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	ctrdversion "demo/over/version"

	api "demo/over/api/services/version/v1"
	"google.golang.org/grpc"
)

var _ api.VersionServer = &service{}

func init() {
	plugin.Register(&plugin.Registration{
		Type:   plugin.GRPCPlugin,
		ID:     "version",
		InitFn: initFunc,
	})
}

func initFunc(ic *plugin.InitContext) (interface{}, error) {
	return &service{}, nil
}

type service struct {
	api.UnimplementedVersionServer
}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterVersionServer(server, s)
	return nil
}

func (s *service) Version(ctx context.Context, _ *ptypes.Empty) (*api.VersionResponse, error) {
	return &api.VersionResponse{
		Version:  ctrdversion.Version,
		Revision: ctrdversion.Revision,
	}, nil
}
