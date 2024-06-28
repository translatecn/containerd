package proxy

import (
	"context"
	api "demo/pkg/api/services/sandbox/v1"
	"demo/pkg/errdefs"
	"demo/pkg/platforms"
	"demo/pkg/sandbox"
	"google.golang.org/protobuf/types/known/anypb"
)

// remoteSandboxController is a low level GRPC client for containerd's sandbox controller service
type remoteSandboxController struct {
	client api.ControllerClient
}

var _ sandbox.Controller = (*remoteSandboxController)(nil)

// NewSandboxController creates a client for a sandbox controller
func NewSandboxController(client api.ControllerClient) sandbox.Controller {
	return &remoteSandboxController{client: client}
}

func (s *remoteSandboxController) Create(ctx context.Context, sandboxID string, opts ...sandbox.CreateOpt) error {
	var options sandbox.CreateOptions
	for _, opt := range opts {
		opt(&options)
	}
	_, err := s.client.Create(ctx, &api.ControllerCreateRequest{
		SandboxID: sandboxID,
		Rootfs:    options.Rootfs,
		Options: &anypb.Any{
			TypeUrl: options.Options.GetTypeUrl(),
			Value:   options.Options.GetValue(),
		},
		NetnsPath: options.NetNSPath,
	})
	if err != nil {
		return errdefs.FromGRPC(err)
	}

	return nil
}

func (s *remoteSandboxController) Start(ctx context.Context, sandboxID string) (sandbox.ControllerInstance, error) {
	resp, err := s.client.Start(ctx, &api.ControllerStartRequest{SandboxID: sandboxID})
	if err != nil {
		return sandbox.ControllerInstance{}, errdefs.FromGRPC(err)
	}

	return sandbox.ControllerInstance{
		SandboxID: sandboxID,
		Pid:       resp.GetPid(),
		CreatedAt: resp.GetCreatedAt().AsTime(),
		Labels:    resp.GetLabels(),
	}, nil
}

func (s *remoteSandboxController) Platform(ctx context.Context, sandboxID string) (platforms.Platform, error) {
	resp, err := s.client.Platform(ctx, &api.ControllerPlatformRequest{SandboxID: sandboxID})
	if err != nil {
		return platforms.Platform{}, errdefs.FromGRPC(err)
	}

	platform := resp.GetPlatform()
	return platforms.Platform{
		Architecture: platform.GetArchitecture(),
		OS:           platform.GetOS(),
		Variant:      platform.GetVariant(),
	}, nil
}

func (s *remoteSandboxController) Stop(ctx context.Context, sandboxID string, opts ...sandbox.StopOpt) error {
	var soptions sandbox.StopOptions
	for _, opt := range opts {
		opt(&soptions)
	}
	req := &api.ControllerStopRequest{SandboxID: sandboxID}
	if soptions.Timeout != nil {
		req.TimeoutSecs = uint32(soptions.Timeout.Seconds())
	}
	_, err := s.client.Stop(ctx, req)
	if err != nil {
		return errdefs.FromGRPC(err)
	}

	return nil
}

func (s *remoteSandboxController) Shutdown(ctx context.Context, sandboxID string) error {
	_, err := s.client.Shutdown(ctx, &api.ControllerShutdownRequest{SandboxID: sandboxID})
	if err != nil {
		return errdefs.FromGRPC(err)
	}

	return nil
}

func (s *remoteSandboxController) Wait(ctx context.Context, sandboxID string) (sandbox.ExitStatus, error) {
	resp, err := s.client.Wait(ctx, &api.ControllerWaitRequest{SandboxID: sandboxID})
	if err != nil {
		return sandbox.ExitStatus{}, errdefs.FromGRPC(err)
	}

	return sandbox.ExitStatus{
		ExitStatus: resp.GetExitStatus(),
		ExitedAt:   resp.GetExitedAt().AsTime(),
	}, nil
}

func (s *remoteSandboxController) Status(ctx context.Context, sandboxID string, verbose bool) (sandbox.ControllerStatus, error) {
	resp, err := s.client.Status(ctx, &api.ControllerStatusRequest{SandboxID: sandboxID, Verbose: verbose})
	if err != nil {
		return sandbox.ControllerStatus{}, errdefs.FromGRPC(err)
	}
	return sandbox.ControllerStatus{
		SandboxID: sandboxID,
		Pid:       resp.GetPid(),
		State:     resp.GetState(),
		Info:      resp.GetInfo(),
		CreatedAt: resp.GetCreatedAt().AsTime(),
		ExitedAt:  resp.GetExitedAt().AsTime(),
		Extra:     resp.GetExtra(),
	}, nil
}
