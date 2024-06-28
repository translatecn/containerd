package proxy

import (
	"context"
	api "demo/pkg/api/services/sandbox/v1"
	"demo/pkg/errdefs"
	"demo/pkg/sandbox"
)

// remoteSandboxStore is a low-level containerd client to manage sandbox environments metadata
type remoteSandboxStore struct {
	client api.StoreClient
}

var _ sandbox.Store = (*remoteSandboxStore)(nil)

// NewSandboxStore create a client for a sandbox store
func NewSandboxStore(client api.StoreClient) sandbox.Store {
	return &remoteSandboxStore{client: client}
}

func (s *remoteSandboxStore) Create(ctx context.Context, sandbox2 sandbox.Sandbox) (sandbox.Sandbox, error) {
	resp, err := s.client.Create(ctx, &api.StoreCreateRequest{
		Sandbox: sandbox.ToProto(&sandbox2),
	})
	if err != nil {
		return sandbox.Sandbox{}, errdefs.FromGRPC(err)
	}

	return sandbox.FromProto(resp.Sandbox), nil
}

func (s *remoteSandboxStore) Update(ctx context.Context, sandbox2 sandbox.Sandbox, fieldpaths ...string) (sandbox.Sandbox, error) {
	resp, err := s.client.Update(ctx, &api.StoreUpdateRequest{
		Sandbox: sandbox.ToProto(&sandbox2),
		Fields:  fieldpaths,
	})
	if err != nil {
		return sandbox.Sandbox{}, errdefs.FromGRPC(err)
	}

	return sandbox.FromProto(resp.Sandbox), nil
}

func (s *remoteSandboxStore) Get(ctx context.Context, id string) (sandbox.Sandbox, error) {
	resp, err := s.client.Get(ctx, &api.StoreGetRequest{
		SandboxID: id,
	})
	if err != nil {
		return sandbox.Sandbox{}, errdefs.FromGRPC(err)
	}

	return sandbox.FromProto(resp.Sandbox), nil
}

func (s *remoteSandboxStore) List(ctx context.Context, filters ...string) ([]sandbox.Sandbox, error) {
	resp, err := s.client.List(ctx, &api.StoreListRequest{
		Filters: filters,
	})
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	out := make([]sandbox.Sandbox, len(resp.List))
	for i := range resp.List {
		out[i] = sandbox.FromProto(resp.List[i])
	}

	return out, nil
}

func (s *remoteSandboxStore) Delete(ctx context.Context, id string) error {
	_, err := s.client.Delete(ctx, &api.StoreDeleteRequest{
		SandboxID: id,
	})

	return errdefs.FromGRPC(err)
}
