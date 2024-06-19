package containerd

import (
	"context"
	"demo/over/namespaces"
	"demo/over/protobuf/types"
	"strings"

	api "demo/over/api/services/namespaces/v1"
	"demo/over/errdefs"
)

// NewNamespaceStoreFromClient returns a new namespace store
func NewNamespaceStoreFromClient(client api.NamespacesClient) namespaces.Store {
	return &remoteNamespaces{client: client}
}

type remoteNamespaces struct {
	client api.NamespacesClient
}

func (r *remoteNamespaces) Create(ctx context.Context, namespace string, labels map[string]string) error {
	var req api.CreateNamespaceRequest

	req.Namespace = &api.Namespace{
		Name:   namespace,
		Labels: labels,
	}

	_, err := r.client.Create(ctx, &req)
	if err != nil {
		return errdefs.FromGRPC(err)
	}

	return nil
}

func (r *remoteNamespaces) Labels(ctx context.Context, namespace string) (map[string]string, error) {
	var req api.GetNamespaceRequest
	req.Name = namespace

	resp, err := r.client.Get(ctx, &req)
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	return resp.Namespace.Labels, nil
}

func (r *remoteNamespaces) SetLabel(ctx context.Context, namespace, key, value string) error {
	var req api.UpdateNamespaceRequest

	req.Namespace = &api.Namespace{
		Name:   namespace,
		Labels: map[string]string{key: value},
	}

	req.UpdateMask = &types.FieldMask{
		Paths: []string{strings.Join([]string{"labels", key}, ".")},
	}

	_, err := r.client.Update(ctx, &req)
	if err != nil {
		return errdefs.FromGRPC(err)
	}

	return nil
}

func (r *remoteNamespaces) List(ctx context.Context) ([]string, error) {
	var req api.ListNamespacesRequest

	resp, err := r.client.List(ctx, &req)
	if err != nil {
		return nil, errdefs.FromGRPC(err)
	}

	var namespaces []string

	for _, ns := range resp.Namespaces {
		namespaces = append(namespaces, ns.Name)
	}

	return namespaces, nil
}

func (r *remoteNamespaces) Delete(ctx context.Context, namespace string, opts ...namespaces.DeleteOpts) error {
	i := namespaces.DeleteInfo{
		Name: namespace,
	}
	for _, o := range opts {
		if err := o(ctx, &i); err != nil {
			return err
		}
	}
	req := api.DeleteNamespaceRequest{
		Name: namespace,
	}
	_, err := r.client.Delete(ctx, &req)
	if err != nil {
		return errdefs.FromGRPC(err)
	}

	return nil
}
