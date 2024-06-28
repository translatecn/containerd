package namespaces

import (
	"context"

	"demo/pkg/ttrpc"
)

const (
	// TTRPCHeader defines the header name for specifying a containerd namespace
	TTRPCHeader = "containerd-namespace-ttrpc"
)

func copyMetadata(src ttrpc.MD) ttrpc.MD {
	md := ttrpc.MD{}
	for k, v := range src {
		md[k] = append(md[k], v...)
	}
	return md
}

func withTTRPCNamespaceHeader(ctx context.Context, namespace string) context.Context {
	md, ok := ttrpc.GetMetadata(ctx)
	if !ok {
		md = ttrpc.MD{}
	} else {
		md = copyMetadata(md)
	}
	md.Set(TTRPCHeader, namespace)
	return ttrpc.WithMetadata(ctx, md)
}

func fromTTRPCHeader(ctx context.Context) (string, bool) {
	return ttrpc.GetMetadataValue(ctx, TTRPCHeader)
}
