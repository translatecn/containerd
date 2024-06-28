package namespaces

import (
	"context"
	"demo/pkg/errdefs"
	"demo/pkg/identifiers"
	"fmt"
)

const (
	// NamespaceEnvVar is the environment variable key name
	NamespaceEnvVar = "CONTAINERD_NAMESPACE"
	// Default is the name of the default namespace
	Default = "default"
)

type namespaceKey struct{}

// WithNamespace sets a given namespace on the context
func WithNamespace(ctx context.Context, namespace string) context.Context {
	ctx = context.WithValue(ctx, namespaceKey{}, namespace) // set our key for namespace
	// also store on the grpc and ttrpc headers so it gets picked up by any clients that
	// are using this.
	return withTTRPCNamespaceHeader(withGRPCNamespaceHeader(ctx, namespace), namespace)
}

// Namespace returns the namespace from the context.
//
// The namespace is not guaranteed to be valid.
func Namespace(ctx context.Context) (string, bool) {
	namespace, ok := ctx.Value(namespaceKey{}).(string)
	if !ok {
		if namespace, ok = fromGRPCHeader(ctx); !ok {
			return fromTTRPCHeader(ctx)
		}
	}
	return namespace, ok
}

// NamespaceRequired returns the valid namespace from the context or an error.
func NamespaceRequired(ctx context.Context) (string, error) {
	namespace, ok := Namespace(ctx)
	if !ok || namespace == "" {
		return "", fmt.Errorf("namespace is required: %w", errdefs.ErrFailedPrecondition)
	}
	if err := identifiers.Validate(namespace); err != nil {
		return "", fmt.Errorf("namespace validation: %w", err)
	}
	return namespace, nil
}
