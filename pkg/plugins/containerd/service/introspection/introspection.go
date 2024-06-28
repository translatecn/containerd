package introspection

import (
	context "context"
	api "demo/pkg/api/services/introspection/v1"
	"demo/pkg/errdefs"
	"demo/pkg/log"
	ptypes "demo/pkg/protobuf/types"
)

// Service defines the introspection service interface
type Service interface {
	Plugins(context.Context, []string) (*api.PluginsResponse, error)
	Server(context.Context, *ptypes.Empty) (*api.ServerResponse, error)
}

type introspectionRemote struct {
	client api.IntrospectionClient
}

var _ = (Service)(&introspectionRemote{})

// NewIntrospectionServiceFromClient creates a new introspection service from an API client
func NewIntrospectionServiceFromClient(c api.IntrospectionClient) Service {
	return &introspectionRemote{client: c}
}

func (i *introspectionRemote) Plugins(ctx context.Context, filters []string) (*api.PluginsResponse, error) {
	log.G(ctx).WithField("filters", filters).Debug("remote introspection plugin filters")
	resp, err := i.client.Plugins(ctx, &api.PluginsRequest{
		Filters: filters,
	})

	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	return resp, nil
}

func (i *introspectionRemote) Server(ctx context.Context, in *ptypes.Empty) (*api.ServerResponse, error) {
	resp, err := i.client.Server(ctx, in)

	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	return resp, nil
}
