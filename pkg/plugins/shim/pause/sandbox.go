package pause

import (
	"context"
	"demo/pkg/plugin"
	"demo/pkg/shutdown"
	"runtime"

	"demo/pkg/api/types"
	"demo/pkg/ttrpc"
	log "github.com/sirupsen/logrus"

	api "demo/pkg/api/runtime/sandbox/v1"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.TTRPCPlugin,
		ID:   "pause",
		Requires: []plugin.Type{
			plugin.InternalPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			ss, err := ic.GetByID(plugin.InternalPlugin, "shutdown")
			if err != nil {
				return nil, err
			}

			return &PauseService{
				shutdown: ss.(shutdown.Service),
			}, nil
		},
	})
}

type PauseService struct {
	shutdown shutdown.Service
}

var _ api.TTRPCSandboxService = (*PauseService)(nil)

func (p *PauseService) RegisterTTRPC(server *ttrpc.Server) error {
	api.RegisterTTRPCSandboxService(server, p)
	return nil
}

func (p *PauseService) CreateSandbox(ctx context.Context, req *api.CreateSandboxRequest) (*api.CreateSandboxResponse, error) {
	log.Debugf("create sandbox request: %+v", req)
	return &api.CreateSandboxResponse{}, nil
}

func (p *PauseService) StartSandbox(ctx context.Context, req *api.StartSandboxRequest) (*api.StartSandboxResponse, error) {
	log.Debugf("start sandbox request: %+v", req)
	return &api.StartSandboxResponse{}, nil
}

func (p *PauseService) Platform(ctx context.Context, req *api.PlatformRequest) (*api.PlatformResponse, error) {
	log.Debugf("platform request: %+v", req)

	platform := types.Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	return &api.PlatformResponse{Platform: &platform}, nil
}

func (p *PauseService) StopSandbox(ctx context.Context, req *api.StopSandboxRequest) (*api.StopSandboxResponse, error) {
	log.Debugf("stop sandbox request: %+v", req)
	p.shutdown.Shutdown()
	return &api.StopSandboxResponse{}, nil
}

func (p *PauseService) WaitSandbox(ctx context.Context, req *api.WaitSandboxRequest) (*api.WaitSandboxResponse, error) {
	return &api.WaitSandboxResponse{
		ExitStatus: 0,
	}, nil
}

func (p *PauseService) SandboxStatus(ctx context.Context, req *api.SandboxStatusRequest) (*api.SandboxStatusResponse, error) {
	log.Debugf("sandbox status request: %+v", req)
	return &api.SandboxStatusResponse{}, nil
}

func (p *PauseService) PingSandbox(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	return &api.PingResponse{}, nil
}

func (p *PauseService) ShutdownSandbox(ctx context.Context, request *api.ShutdownSandboxRequest) (*api.ShutdownSandboxResponse, error) {
	return &api.ShutdownSandboxResponse{}, nil
}
