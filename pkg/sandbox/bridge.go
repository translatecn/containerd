package sandbox

import (
	"context"
	"fmt"

	"demo/pkg/ttrpc"
	"google.golang.org/grpc"

	api "demo/pkg/api/runtime/sandbox/v1"
)

// NewClient returns a new sandbox client that handles both GRPC and TTRPC clients.
func NewClient(client interface{}) (api.TTRPCSandboxService, error) {
	switch c := client.(type) {
	case *ttrpc.Client:
		return api.NewTTRPCSandboxClient(c), nil
	case grpc.ClientConnInterface:
		return &grpcBridge{api.NewSandboxClient(c)}, nil
	default:
		return nil, fmt.Errorf("unsupported client type %T", client)
	}
}

type grpcBridge struct {
	client api.SandboxClient
}

var _ api.TTRPCSandboxService = (*grpcBridge)(nil)

func (g *grpcBridge) CreateSandbox(ctx context.Context, request *api.CreateSandboxRequest) (*api.CreateSandboxResponse, error) {
	return g.client.CreateSandbox(ctx, request)
}

func (g *grpcBridge) StartSandbox(ctx context.Context, request *api.StartSandboxRequest) (*api.StartSandboxResponse, error) {
	return g.client.StartSandbox(ctx, request)
}

func (g *grpcBridge) Platform(ctx context.Context, request *api.PlatformRequest) (*api.PlatformResponse, error) {
	return g.client.Platform(ctx, request)
}

func (g *grpcBridge) StopSandbox(ctx context.Context, request *api.StopSandboxRequest) (*api.StopSandboxResponse, error) {
	return g.client.StopSandbox(ctx, request)
}

func (g *grpcBridge) WaitSandbox(ctx context.Context, request *api.WaitSandboxRequest) (*api.WaitSandboxResponse, error) {
	return g.client.WaitSandbox(ctx, request)
}

func (g *grpcBridge) SandboxStatus(ctx context.Context, request *api.SandboxStatusRequest) (*api.SandboxStatusResponse, error) {
	return g.client.SandboxStatus(ctx, request)
}

func (g *grpcBridge) PingSandbox(ctx context.Context, request *api.PingRequest) (*api.PingResponse, error) {
	return g.client.PingSandbox(ctx, request)
}

func (g *grpcBridge) ShutdownSandbox(ctx context.Context, request *api.ShutdownSandboxRequest) (*api.ShutdownSandboxResponse, error) {
	return g.client.ShutdownSandbox(ctx, request)
}
