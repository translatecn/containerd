package namespaces

import (
	"context"
	metadata2 "demo/pkg/metadata"
	"demo/pkg/namespaces"
	"demo/pkg/plugin"
	"demo/pkg/plugins"
	ptypes "demo/pkg/protobuf/types"
	"strings"

	eventstypes "demo/pkg/api/events"
	api "demo/pkg/api/services/namespaces/v1"
	"demo/pkg/errdefs"
	"demo/pkg/events"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.ServicePlugin,
		ID:   plugins.NamespacesService,
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
			return &local{
				db:        m.(*metadata2.DB),
				publisher: ep.(events.Publisher),
			}, nil
		},
	})
}

// Provide local namespaces service instead of local namespace store,
// because namespace store interface doesn't provide enough functionality
// for namespaces service.
type local struct {
	db        *metadata2.DB
	publisher events.Publisher
}

var _ api.NamespacesClient = &local{}

func (l *local) Get(ctx context.Context, req *api.GetNamespaceRequest, _ ...grpc.CallOption) (*api.GetNamespaceResponse, error) {
	var resp api.GetNamespaceResponse

	return &resp, l.withStoreView(ctx, func(ctx context.Context, store namespaces.Store) error {
		labels, err := store.Labels(ctx, req.Name)
		if err != nil {
			return errdefs.ToGRPC(err)
		}

		resp.Namespace = &api.Namespace{
			Name:   req.Name,
			Labels: labels,
		}

		return nil
	})
}

func (l *local) List(ctx context.Context, req *api.ListNamespacesRequest, _ ...grpc.CallOption) (*api.ListNamespacesResponse, error) {
	var resp api.ListNamespacesResponse

	return &resp, l.withStoreView(ctx, func(ctx context.Context, store namespaces.Store) error {
		namespaces, err := store.List(ctx)
		if err != nil {
			return err
		}

		for _, namespace := range namespaces {
			labels, err := store.Labels(ctx, namespace)
			if err != nil {
				// In general, this should be unlikely, since we are holding a
				// transaction to service this request.
				return errdefs.ToGRPC(err)
			}

			resp.Namespaces = append(resp.Namespaces, &api.Namespace{
				Name:   namespace,
				Labels: labels,
			})
		}

		return nil
	})
}

func (l *local) Create(ctx context.Context, req *api.CreateNamespaceRequest, _ ...grpc.CallOption) (*api.CreateNamespaceResponse, error) {
	var resp api.CreateNamespaceResponse

	if err := l.withStoreUpdate(ctx, func(ctx context.Context, store namespaces.Store) error {
		if err := store.Create(ctx, req.Namespace.Name, req.Namespace.Labels); err != nil {
			return errdefs.ToGRPC(err)
		}

		for k, v := range req.Namespace.Labels {
			if err := store.SetLabel(ctx, req.Namespace.Name, k, v); err != nil {
				return err
			}
		}

		resp.Namespace = req.Namespace
		return nil
	}); err != nil {
		return &resp, err
	}

	ctx = namespaces.WithNamespace(ctx, req.Namespace.Name)
	if err := l.publisher.Publish(ctx, "/namespaces/create", &eventstypes.NamespaceCreate{
		Name:   req.Namespace.Name,
		Labels: req.Namespace.Labels,
	}); err != nil {
		return &resp, err
	}

	return &resp, nil

}

func (l *local) Update(ctx context.Context, req *api.UpdateNamespaceRequest, _ ...grpc.CallOption) (*api.UpdateNamespaceResponse, error) {
	var resp api.UpdateNamespaceResponse
	if err := l.withStoreUpdate(ctx, func(ctx context.Context, store namespaces.Store) error {
		if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
			for _, path := range req.UpdateMask.Paths {
				switch {
				case strings.HasPrefix(path, "labels."):
					key := strings.TrimPrefix(path, "labels.")
					if err := store.SetLabel(ctx, req.Namespace.Name, key, req.Namespace.Labels[key]); err != nil {
						return err
					}
				default:
					return status.Errorf(codes.InvalidArgument, "cannot update %q field", path)
				}
			}
		} else {
			// clear out the existing labels and then set them to the incoming request.
			// get current set of labels
			labels, err := store.Labels(ctx, req.Namespace.Name)
			if err != nil {
				return errdefs.ToGRPC(err)
			}

			for k := range labels {
				if err := store.SetLabel(ctx, req.Namespace.Name, k, ""); err != nil {
					return err
				}
			}

			for k, v := range req.Namespace.Labels {
				if err := store.SetLabel(ctx, req.Namespace.Name, k, v); err != nil {
					return err
				}

			}
		}

		return nil
	}); err != nil {
		return &resp, err
	}

	ctx = namespaces.WithNamespace(ctx, req.Namespace.Name)
	if err := l.publisher.Publish(ctx, "/namespaces/update", &eventstypes.NamespaceUpdate{
		Name:   req.Namespace.Name,
		Labels: req.Namespace.Labels,
	}); err != nil {
		return &resp, err
	}

	return &resp, nil
}

func (l *local) Delete(ctx context.Context, req *api.DeleteNamespaceRequest, _ ...grpc.CallOption) (*ptypes.Empty, error) {
	if err := l.withStoreUpdate(ctx, func(ctx context.Context, store namespaces.Store) error {
		return errdefs.ToGRPC(store.Delete(ctx, req.Name))
	}); err != nil {
		return &ptypes.Empty{}, err
	}
	// set the namespace in the context before publishing the event
	ctx = namespaces.WithNamespace(ctx, req.Name)
	if err := l.publisher.Publish(ctx, "/namespaces/delete", &eventstypes.NamespaceDelete{
		Name: req.Name,
	}); err != nil {
		return &ptypes.Empty{}, err
	}

	return &ptypes.Empty{}, nil
}

func (l *local) withStore(ctx context.Context, fn func(ctx context.Context, store namespaces.Store) error) func(tx *bolt.Tx) error {
	return func(tx *bolt.Tx) error { return fn(ctx, metadata2.NewNamespaceStore(tx)) }
}

func (l *local) withStoreView(ctx context.Context, fn func(ctx context.Context, store namespaces.Store) error) error {
	return l.db.View(l.withStore(ctx, fn))
}

func (l *local) withStoreUpdate(ctx context.Context, fn func(ctx context.Context, store namespaces.Store) error) error {
	return l.db.Update(l.withStore(ctx, fn))
}
