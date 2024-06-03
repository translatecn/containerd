/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package sandbox

import (
	"context"
	"demo/others/log"
	over_plugin2 "demo/over/plugin"
	sandbox2 "demo/pkg/sandbox"
	"google.golang.org/grpc"

	"demo/over/errdefs"
	api "demo/pkg/api/services/sandbox/v1"
	"demo/pkg/api/types"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.GRPCPlugin,
		ID:   "sandboxes",
		Requires: []over_plugin2.Type{
			over_plugin2.SandboxStorePlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			sp, err := ic.GetByID(over_plugin2.SandboxStorePlugin, "local")
			if err != nil {
				return nil, err
			}

			return &sandboxService{store: sp.(sandbox2.Store)}, nil
		},
	})
}

type sandboxService struct {
	store sandbox2.Store
	api.UnimplementedStoreServer
}

var _ api.StoreServer = (*sandboxService)(nil)

func (s *sandboxService) Register(server *grpc.Server) error {
	api.RegisterStoreServer(server, s)
	return nil
}

func (s *sandboxService) Create(ctx context.Context, req *api.StoreCreateRequest) (*api.StoreCreateResponse, error) {
	log.G(ctx).WithField("req", req).Debug("create sandbox")
	sb, err := s.store.Create(ctx, sandbox2.FromProto(req.Sandbox))
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}

	return &api.StoreCreateResponse{Sandbox: sandbox2.ToProto(&sb)}, nil
}

func (s *sandboxService) Update(ctx context.Context, req *api.StoreUpdateRequest) (*api.StoreUpdateResponse, error) {
	log.G(ctx).WithField("req", req).Debug("update sandbox")

	sb, err := s.store.Update(ctx, sandbox2.FromProto(req.Sandbox), req.Fields...)
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}

	return &api.StoreUpdateResponse{Sandbox: sandbox2.ToProto(&sb)}, nil
}

func (s *sandboxService) List(ctx context.Context, req *api.StoreListRequest) (*api.StoreListResponse, error) {
	log.G(ctx).WithField("req", req).Debug("list sandboxes")

	resp, err := s.store.List(ctx, req.Filters...)
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}

	list := make([]*types.Sandbox, len(resp))
	for i := range resp {
		list[i] = sandbox2.ToProto(&resp[i])
	}

	return &api.StoreListResponse{List: list}, nil
}

func (s *sandboxService) Get(ctx context.Context, req *api.StoreGetRequest) (*api.StoreGetResponse, error) {
	log.G(ctx).WithField("req", req).Debug("get sandbox")
	resp, err := s.store.Get(ctx, req.SandboxID)
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}

	desc := sandbox2.ToProto(&resp)
	return &api.StoreGetResponse{Sandbox: desc}, nil
}

func (s *sandboxService) Delete(ctx context.Context, req *api.StoreDeleteRequest) (*api.StoreDeleteResponse, error) {
	log.G(ctx).WithField("req", req).Debug("delete sandbox")
	if err := s.store.Delete(ctx, req.SandboxID); err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}

	return &api.StoreDeleteResponse{}, nil
}
