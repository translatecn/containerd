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

package namespaces

import (
	"context"
	over_plugin2 "demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	"errors"

	api "demo/pkg/api/services/namespaces/v1"
	"demo/services"
	"google.golang.org/grpc"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.GRPCPlugin,
		ID:   "namespaces",
		Requires: []over_plugin2.Type{
			over_plugin2.ServicePlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			plugins, err := ic.GetByType(over_plugin2.ServicePlugin)
			if err != nil {
				return nil, err
			}
			p, ok := plugins[services.NamespacesService]
			if !ok {
				return nil, errors.New("namespaces service not found")
			}
			i, err := p.Instance()
			if err != nil {
				return nil, err
			}
			return &service{local: i.(api.NamespacesClient)}, nil
		},
	})
}

type service struct {
	local api.NamespacesClient
	api.UnimplementedNamespacesServer
}

var _ api.NamespacesServer = &service{}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterNamespacesServer(server, s)
	return nil
}

func (s *service) Get(ctx context.Context, req *api.GetNamespaceRequest) (*api.GetNamespaceResponse, error) {
	return s.local.Get(ctx, req)
}

func (s *service) List(ctx context.Context, req *api.ListNamespacesRequest) (*api.ListNamespacesResponse, error) {
	return s.local.List(ctx, req)
}

func (s *service) Create(ctx context.Context, req *api.CreateNamespaceRequest) (*api.CreateNamespaceResponse, error) {
	return s.local.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, req *api.UpdateNamespaceRequest) (*api.UpdateNamespaceResponse, error) {
	return s.local.Update(ctx, req)
}

func (s *service) Delete(ctx context.Context, req *api.DeleteNamespaceRequest) (*ptypes.Empty, error) {
	return s.local.Delete(ctx, req)
}
