package service

import (
	"context"
	metadata2 "demo/over/metadata"
	"demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	"demo/plugins"
	"io"

	eventstypes "demo/over/api/events"
	api "demo/over/api/services/containers/v1"
	"demo/over/containers"
	"demo/over/errdefs"
	"demo/over/events"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcm "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.ServicePlugin,
		ID:   plugins.ContainersService,
		Requires: []plugin.Type{
			plugin.EventPlugin,
			plugin.MetadataPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			ep, err := ic.Get(plugin.EventPlugin)
			if err != nil {
				return nil, err
			}
			db := m.(*metadata2.DB)
			return &localContainers{
				Store:     metadata2.NewContainerStore(db),
				db:        db,
				publisher: ep.(events.Publisher),
			}, nil
		},
	})
}

type localContainers struct {
	containers.Store
	db        *metadata2.DB
	publisher events.Publisher
}

var _ api.ContainersClient = &localContainers{}

func (l *localContainers) Get(ctx context.Context, req *api.GetContainerRequest, _ ...grpc.CallOption) (*api.GetContainerResponse, error) {
	var resp api.GetContainerResponse

	return &resp, errdefs.ToGRPC(l.withStoreView(ctx, func(ctx context.Context) error {
		container, err := l.Store.Get(ctx, req.ID)
		if err != nil {
			return err
		}
		containerpb := containerToProto(&container)
		resp.Container = containerpb

		return nil
	}))
}

func (l *localContainers) List(ctx context.Context, req *api.ListContainersRequest, _ ...grpc.CallOption) (*api.ListContainersResponse, error) {
	var resp api.ListContainersResponse
	return &resp, errdefs.ToGRPC(l.withStoreView(ctx, func(ctx context.Context) error {
		containers, err := l.Store.List(ctx, req.Filters...)
		if err != nil {
			return err
		}
		resp.Containers = containersToProto(containers)
		return nil
	}))
}

func (l *localContainers) ListStream(ctx context.Context, req *api.ListContainersRequest, _ ...grpc.CallOption) (api.Containers_ListStreamClient, error) {
	stream := &localContainersStream{
		ctx: ctx,
	}
	return stream, errdefs.ToGRPC(l.withStoreView(ctx, func(ctx context.Context) error {
		containers, err := l.Store.List(ctx, req.Filters...)
		if err != nil {
			return err
		}
		stream.containers = containersToProto(containers)
		return nil
	}))
}

func (l *localContainers) Create(ctx context.Context, req *api.CreateContainerRequest, _ ...grpc.CallOption) (*api.CreateContainerResponse, error) {
	var resp api.CreateContainerResponse

	if err := l.withStoreUpdate(ctx, func(ctx context.Context) error {
		container := containerFromProto(req.Container)

		created, err := l.Store.Create(ctx, container)
		if err != nil {
			return err
		}

		resp.Container = containerToProto(&created)

		return nil
	}); err != nil {
		return &resp, errdefs.ToGRPC(err)
	}
	if err := l.publisher.Publish(ctx, "/containers/create", &eventstypes.ContainerCreate{
		ID:    resp.Container.ID,
		Image: resp.Container.Image,
		Runtime: &eventstypes.ContainerCreate_Runtime{
			Name:    resp.Container.Runtime.Name,
			Options: resp.Container.Runtime.Options,
		},
	}); err != nil {
		return &resp, err
	}

	return &resp, nil
}

func (l *localContainers) Update(ctx context.Context, req *api.UpdateContainerRequest, _ ...grpc.CallOption) (*api.UpdateContainerResponse, error) {
	if req.Container.ID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Container.ID required")
	}
	var (
		resp      api.UpdateContainerResponse
		container = containerFromProto(req.Container)
	)

	if err := l.withStoreUpdate(ctx, func(ctx context.Context) error {
		var fieldpaths []string
		if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
			fieldpaths = append(fieldpaths, req.UpdateMask.Paths...)
		}

		updated, err := l.Store.Update(ctx, container, fieldpaths...)
		if err != nil {
			return err
		}

		resp.Container = containerToProto(&updated)
		return nil
	}); err != nil {
		return &resp, errdefs.ToGRPC(err)
	}

	if err := l.publisher.Publish(ctx, "/containers/update", &eventstypes.ContainerUpdate{
		ID:          resp.Container.ID,
		Image:       resp.Container.Image,
		Labels:      resp.Container.Labels,
		SnapshotKey: resp.Container.SnapshotKey,
	}); err != nil {
		return &resp, err
	}

	return &resp, nil
}

func (l *localContainers) Delete(ctx context.Context, req *api.DeleteContainerRequest, _ ...grpc.CallOption) (*ptypes.Empty, error) {
	if err := l.withStoreUpdate(ctx, func(ctx context.Context) error {
		return l.Store.Delete(ctx, req.ID)
	}); err != nil {
		return &ptypes.Empty{}, errdefs.ToGRPC(err)
	}

	if err := l.publisher.Publish(ctx, "/containers/delete", &eventstypes.ContainerDelete{
		ID: req.ID,
	}); err != nil {
		return &ptypes.Empty{}, err
	}

	return &ptypes.Empty{}, nil
}

func (l *localContainers) withStore(ctx context.Context, fn func(ctx context.Context) error) func(tx *bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		return fn(metadata2.WithTransactionContext(ctx, tx))
	}
}

func (l *localContainers) withStoreView(ctx context.Context, fn func(ctx context.Context) error) error {
	return l.db.View(l.withStore(ctx, fn))
}

func (l *localContainers) withStoreUpdate(ctx context.Context, fn func(ctx context.Context) error) error {
	return l.db.Update(l.withStore(ctx, fn))
}

type localContainersStream struct {
	ctx        context.Context
	containers []*api.Container
	i          int
}

func (s *localContainersStream) Recv() (*api.ListContainerMessage, error) {
	if s.i >= len(s.containers) {
		return nil, io.EOF
	}
	c := s.containers[s.i]
	s.i++
	return &api.ListContainerMessage{
		Container: c,
	}, nil
}

func (s *localContainersStream) Context() context.Context {
	return s.ctx
}

func (s *localContainersStream) CloseSend() error {
	return nil
}

func (s *localContainersStream) Header() (grpcm.MD, error) {
	return nil, nil
}

func (s *localContainersStream) Trailer() grpcm.MD {
	return nil
}

func (s *localContainersStream) SendMsg(m interface{}) error {
	return nil
}

func (s *localContainersStream) RecvMsg(m interface{}) error {
	return nil
}
