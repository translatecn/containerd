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

package proxy

import (
	"context"
	sandbox2 "demo/pkg/sandbox"

	"demo/over/errdefs"
	api "demo/pkg/api/services/sandbox/v1"
)

// remoteSandboxStore is a low-level containerd client to manage sandbox environments metadata
type remoteSandboxStore struct {
	client api.StoreClient
}

var _ sandbox2.Store = (*remoteSandboxStore)(nil)

// NewSandboxStore create a client for a sandbox store
func NewSandboxStore(client api.StoreClient) sandbox2.Store {
	return &remoteSandboxStore{client: client}
}

func (s *remoteSandboxStore) Create(ctx context.Context, sandbox sandbox2.Sandbox) (sandbox2.Sandbox, error) {
	resp, err := s.client.Create(ctx, &api.StoreCreateRequest{
		Sandbox: sandbox2.ToProto(&sandbox),
	})
	if err != nil {
		return sandbox2.Sandbox{}, over_errdefs.FromGRPC(err)
	}

	return sandbox2.FromProto(resp.Sandbox), nil
}

func (s *remoteSandboxStore) Update(ctx context.Context, sandbox sandbox2.Sandbox, fieldpaths ...string) (sandbox2.Sandbox, error) {
	resp, err := s.client.Update(ctx, &api.StoreUpdateRequest{
		Sandbox: sandbox2.ToProto(&sandbox),
		Fields:  fieldpaths,
	})
	if err != nil {
		return sandbox2.Sandbox{}, over_errdefs.FromGRPC(err)
	}

	return sandbox2.FromProto(resp.Sandbox), nil
}

func (s *remoteSandboxStore) Get(ctx context.Context, id string) (sandbox2.Sandbox, error) {
	resp, err := s.client.Get(ctx, &api.StoreGetRequest{
		SandboxID: id,
	})
	if err != nil {
		return sandbox2.Sandbox{}, over_errdefs.FromGRPC(err)
	}

	return sandbox2.FromProto(resp.Sandbox), nil
}

func (s *remoteSandboxStore) List(ctx context.Context, filters ...string) ([]sandbox2.Sandbox, error) {
	resp, err := s.client.List(ctx, &api.StoreListRequest{
		Filters: filters,
	})
	if err != nil {
		return nil, over_errdefs.FromGRPC(err)
	}

	out := make([]sandbox2.Sandbox, len(resp.List))
	for i := range resp.List {
		out[i] = sandbox2.FromProto(resp.List[i])
	}

	return out, nil
}

func (s *remoteSandboxStore) Delete(ctx context.Context, id string) error {
	_, err := s.client.Delete(ctx, &api.StoreDeleteRequest{
		SandboxID: id,
	})

	return over_errdefs.FromGRPC(err)
}
