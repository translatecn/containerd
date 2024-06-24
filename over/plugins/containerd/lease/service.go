package lease

import (
	"context"
	leases2 "demo/over/leases"
	"demo/over/plugin"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"

	api "demo/over/api/services/leases/v1"
	"demo/over/errdefs"
	"google.golang.org/grpc"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "leases",
		Requires: []plugin.Type{
			plugin.LeasePlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			i, err := ic.GetByID(plugin.LeasePlugin, "manager")
			if err != nil {
				return nil, err
			}
			return &service{lm: i.(leases2.Manager)}, nil
		},
	})
}

type service struct {
	lm leases2.Manager
	api.UnimplementedLeasesServer
}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterLeasesServer(server, s)
	return nil
}

func (s *service) Create(ctx context.Context, r *api.CreateRequest) (*api.CreateResponse, error) {
	opts := []leases2.Opt{
		leases2.WithLabels(r.Labels),
	}
	if r.ID == "" {
		opts = append(opts, leases2.WithRandomID())
	} else {
		opts = append(opts, leases2.WithID(r.ID))
	}

	l, err := s.lm.Create(ctx, opts...)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &api.CreateResponse{
		Lease: leaseToGRPC(l),
	}, nil
}

func (s *service) Delete(ctx context.Context, r *api.DeleteRequest) (*ptypes.Empty, error) {
	var opts []leases2.DeleteOpt
	if r.Sync {
		opts = append(opts, leases2.SynchronousDelete)
	}
	if err := s.lm.Delete(ctx, leases2.Lease{
		ID: r.ID,
	}, opts...); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return &ptypes.Empty{}, nil
}

func (s *service) List(ctx context.Context, r *api.ListRequest) (*api.ListResponse, error) {
	l, err := s.lm.List(ctx, r.Filters...)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	apileases := make([]*api.Lease, len(l))
	for i := range l {
		apileases[i] = leaseToGRPC(l[i])
	}

	return &api.ListResponse{
		Leases: apileases,
	}, nil
}

func (s *service) AddResource(ctx context.Context, r *api.AddResourceRequest) (*ptypes.Empty, error) {
	lease := leases2.Lease{
		ID: r.ID,
	}

	if err := s.lm.AddResource(ctx, lease, leases2.Resource{
		ID:   r.Resource.ID,
		Type: r.Resource.Type,
	}); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return &ptypes.Empty{}, nil
}

func (s *service) DeleteResource(ctx context.Context, r *api.DeleteResourceRequest) (*ptypes.Empty, error) {
	lease := leases2.Lease{
		ID: r.ID,
	}

	if err := s.lm.DeleteResource(ctx, lease, leases2.Resource{
		ID:   r.Resource.ID,
		Type: r.Resource.Type,
	}); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return &ptypes.Empty{}, nil
}

func (s *service) ListResources(ctx context.Context, r *api.ListResourcesRequest) (*api.ListResourcesResponse, error) {
	lease := leases2.Lease{
		ID: r.ID,
	}

	rs, err := s.lm.ListResources(ctx, lease)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	apiResources := make([]*api.Resource, 0, len(rs))
	for _, i := range rs {
		apiResources = append(apiResources, &api.Resource{
			ID:   i.ID,
			Type: i.Type,
		})
	}
	return &api.ListResourcesResponse{
		Resources: apiResources,
	}, nil
}

func leaseToGRPC(l leases2.Lease) *api.Lease {
	return &api.Lease{
		ID:        l.ID,
		Labels:    l.Labels,
		CreatedAt: protobuf.ToTimestamp(l.CreatedAt),
	}
}
