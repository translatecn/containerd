package namespaces

import "context"

// Store provides introspection about namespaces.
//
// Note that these are slightly different than other objects, which are record
// oriented. A namespace is really just a name and a set of labels. Objects
// that belong to a namespace are returned when the namespace is assigned to a
// given context.
type Store interface {
	Create(ctx context.Context, namespace string, labels map[string]string) error
	Labels(ctx context.Context, namespace string) (map[string]string, error)
	SetLabel(ctx context.Context, namespace, key, value string) error
	List(ctx context.Context) ([]string, error)

	// Delete removes the namespace. The namespace must be empty to be deleted.
	Delete(ctx context.Context, namespace string, opts ...DeleteOpts) error
}

// DeleteInfo specifies information for the deletion of a namespace
type DeleteInfo struct {
	// Name of the namespace
	Name string
}

// DeleteOpts allows the caller to set options for namespace deletion
type DeleteOpts func(context.Context, *DeleteInfo) error
