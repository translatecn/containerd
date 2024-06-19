package sanbox

import (
	"context"
	"demo/over/log"
	"demo/over/plugin"
	"demo/pkg/sandbox"
	"google.golang.org/grpc"

	api "demo/over/api/services/sandbox/v1"
	"demo/over/api/types"
	"demo/over/errdefs"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "sandboxes",
		Requires: []plugin.Type{
			plugin.SandboxStorePlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			sp, err := ic.GetByID(plugin.SandboxStorePlugin, "local")
			if err != nil {
				return nil, err
			}

			return &sandboxService{store: sp.(sandbox.Store)}, nil
		},
	})
}

type sandboxService struct {
	store sandbox.Store
	api.UnimplementedStoreServer
}

var _ api.StoreServer = (*sandboxService)(nil)

func (s *sandboxService) Register(server *grpc.Server) error {
	api.RegisterStoreServer(server, s)
	return nil
}

func (s *sandboxService) Create(ctx context.Context, req *api.StoreCreateRequest) (*api.StoreCreateResponse, error) {
	log.G(ctx).WithField("req", req).Debug("create sandbox")
	sb, err := s.store.Create(ctx, sandbox.FromProto(req.Sandbox))
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &api.StoreCreateResponse{Sandbox: sandbox.ToProto(&sb)}, nil
}

func (s *sandboxService) Update(ctx context.Context, req *api.StoreUpdateRequest) (*api.StoreUpdateResponse, error) {
	log.G(ctx).WithField("req", req).Debug("update sandbox")

	sb, err := s.store.Update(ctx, sandbox.FromProto(req.Sandbox), req.Fields...)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &api.StoreUpdateResponse{Sandbox: sandbox.ToProto(&sb)}, nil
}

func (s *sandboxService) List(ctx context.Context, req *api.StoreListRequest) (*api.StoreListResponse, error) {
	log.G(ctx).WithField("req", req).Debug("list sandboxes")

	resp, err := s.store.List(ctx, req.Filters...)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	list := make([]*types.Sandbox, len(resp))
	for i := range resp {
		list[i] = sandbox.ToProto(&resp[i])
	}

	return &api.StoreListResponse{List: list}, nil
}

func (s *sandboxService) Get(ctx context.Context, req *api.StoreGetRequest) (*api.StoreGetResponse, error) {
	log.G(ctx).WithField("req", req).Debug("get sandbox")
	resp, err := s.store.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	desc := sandbox.ToProto(&resp)
	return &api.StoreGetResponse{Sandbox: desc}, nil
}

func (s *sandboxService) Delete(ctx context.Context, req *api.StoreDeleteRequest) (*api.StoreDeleteResponse, error) {
	log.G(ctx).WithField("req", req).Debug("delete sandbox")
	if err := s.store.Delete(ctx, req.SandboxID); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &api.StoreDeleteResponse{}, nil
}
