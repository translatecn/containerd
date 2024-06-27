package util

import (
	"context"
	"demo/over/cri/constants"
	"demo/over/namespaces"
	"time"
)

// deferCleanupTimeout is the default timeout for containerd cleanup operations
// in defer.
const deferCleanupTimeout = 1 * time.Minute

// DeferContext returns a context for containerd cleanup operations in defer.
// A default timeout is applied to avoid cleanup operation pending forever.
func DeferContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(NamespacedContext(), deferCleanupTimeout)
}

// NamespacedContext returns a context with kubernetes namespace set.
func NamespacedContext() context.Context {
	return WithNamespace(context.Background())
}

// WithNamespace adds kubernetes namespace to the context.
func WithNamespace(ctx context.Context) context.Context {
	return namespaces.WithNamespace(ctx, constants.K8sContainerdNamespace)
}
