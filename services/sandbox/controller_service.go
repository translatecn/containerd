/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package sandbox

import (
	"context"
	"demo/others/log"
	over_plugin2 "demo/over/plugin"
	"demo/over/protobuf"
	"demo/pkg/sandbox"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"

	"demo/over/errdefs"
	eventtypes "demo/pkg/api/events"
	api "demo/pkg/api/services/sandbox/v1"
	"demo/pkg/events"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.GRPCPlugin,
		ID:   "sandbox-controllers",
		Requires: []over_plugin2.Type{
			over_plugin2.SandboxControllerPlugin,
			over_plugin2.EventPlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			sc, err := ic.GetByID(over_plugin2.SandboxControllerPlugin, "local")
			if err != nil {
				return nil, err
			}

			ep, err := ic.Get(over_plugin2.EventPlugin)
			if err != nil {
				return nil, err
			}

			return &controllerService{
				local:     sc.(sandbox.Controller),
				publisher: ep.(events.Publisher),
			}, nil
		},
	})
}

type controllerService struct {
	local     sandbox.Controller
	publisher events.Publisher
	api.UnimplementedControllerServer
}

var _ api.ControllerServer = (*controllerService)(nil)

func (s *controllerService) Register(server *grpc.Server) error {
	api.RegisterControllerServer(server, s)
	return nil
}

func (s *controllerService) Create(ctx context.Context, req *api.ControllerCreateRequest) (*api.ControllerCreateResponse, error) {
	log.G(ctx).WithField("req", req).Debug("create sandbox")
	// TODO: Rootfs
	err := s.local.Create(ctx, req.GetSandboxID(), sandbox.WithOptions(req.GetOptions()))
	if err != nil {
		return &api.ControllerCreateResponse{}, over_errdefs.ToGRPC(err)
	}

	if err := s.publisher.Publish(ctx, "sandboxes/create", &eventtypes.SandboxCreate{
		SandboxID: req.GetSandboxID(),
	}); err != nil {
		return &api.ControllerCreateResponse{}, over_errdefs.ToGRPC(err)
	}

	return &api.ControllerCreateResponse{
		SandboxID: req.GetSandboxID(),
	}, nil
}

func (s *controllerService) Start(ctx context.Context, req *api.ControllerStartRequest) (*api.ControllerStartResponse, error) {
	log.G(ctx).WithField("req", req).Debug("start sandbox")
	inst, err := s.local.Start(ctx, req.GetSandboxID())
	if err != nil {
		return &api.ControllerStartResponse{}, over_errdefs.ToGRPC(err)
	}

	if err := s.publisher.Publish(ctx, "sandboxes/start", &eventtypes.SandboxStart{
		SandboxID: req.GetSandboxID(),
	}); err != nil {
		return &api.ControllerStartResponse{}, over_errdefs.ToGRPC(err)
	}

	return &api.ControllerStartResponse{
		SandboxID: inst.SandboxID,
		Pid:       inst.Pid,
		CreatedAt: over_protobuf.ToTimestamp(inst.CreatedAt),
		Labels:    inst.Labels,
	}, nil
}

func (s *controllerService) Stop(ctx context.Context, req *api.ControllerStopRequest) (*api.ControllerStopResponse, error) {
	log.G(ctx).WithField("req", req).Debug("delete sandbox")
	return &api.ControllerStopResponse{}, over_errdefs.ToGRPC(s.local.Stop(ctx, req.GetSandboxID()))
}

func (s *controllerService) Wait(ctx context.Context, req *api.ControllerWaitRequest) (*api.ControllerWaitResponse, error) {
	log.G(ctx).WithField("req", req).Debug("wait sandbox")
	exitStatus, err := s.local.Wait(ctx, req.GetSandboxID())
	if err != nil {
		return &api.ControllerWaitResponse{}, over_errdefs.ToGRPC(err)
	}

	if err := s.publisher.Publish(ctx, "sandboxes/exit", &eventtypes.SandboxExit{
		SandboxID:  req.GetSandboxID(),
		ExitStatus: exitStatus.ExitStatus,
		ExitedAt:   over_protobuf.ToTimestamp(exitStatus.ExitedAt),
	}); err != nil {
		return &api.ControllerWaitResponse{}, over_errdefs.ToGRPC(err)
	}

	return &api.ControllerWaitResponse{
		ExitStatus: exitStatus.ExitStatus,
		ExitedAt:   over_protobuf.ToTimestamp(exitStatus.ExitedAt),
	}, nil
}

func (s *controllerService) Status(ctx context.Context, req *api.ControllerStatusRequest) (*api.ControllerStatusResponse, error) {
	log.G(ctx).WithField("req", req).Debug("sandbox status")
	cstatus, err := s.local.Status(ctx, req.GetSandboxID(), req.GetVerbose())
	if err != nil {
		return &api.ControllerStatusResponse{}, over_errdefs.ToGRPC(err)
	}
	extra := &anypb.Any{}
	if cstatus.Extra != nil {
		extra = &anypb.Any{
			TypeUrl: cstatus.Extra.GetTypeUrl(),
			Value:   cstatus.Extra.GetValue(),
		}
	}
	return &api.ControllerStatusResponse{
		SandboxID: cstatus.SandboxID,
		Pid:       cstatus.Pid,
		State:     cstatus.State,
		Info:      cstatus.Info,
		CreatedAt: over_protobuf.ToTimestamp(cstatus.CreatedAt),
		ExitedAt:  over_protobuf.ToTimestamp(cstatus.ExitedAt),
		Extra:     extra,
	}, nil
}

func (s *controllerService) Shutdown(ctx context.Context, req *api.ControllerShutdownRequest) (*api.ControllerShutdownResponse, error) {
	log.G(ctx).WithField("req", req).Debug("shutdown sandbox")
	return &api.ControllerShutdownResponse{}, over_errdefs.ToGRPC(s.local.Shutdown(ctx, req.GetSandboxID()))
}
