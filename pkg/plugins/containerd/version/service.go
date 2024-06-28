package version

import (
	"context"
	"demo/pkg/plugin"
	ptypes "demo/pkg/protobuf/types"
	ctrdversion "demo/pkg/version"

	api "demo/pkg/api/services/version/v1"
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
