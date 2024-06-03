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

package transfer

import (
	"context"
	"demo/others/log"
	"demo/others/typeurl/v2"
	"demo/over/errdefs"
	over_plugin2 "demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	transferapi "demo/pkg/api/services/transfer/v1"
	transferTypes "demo/pkg/api/types/transfer"
	"demo/pkg/streaming"
	"demo/pkg/transfer"
	"demo/pkg/transfer/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.GRPCPlugin,
		ID:   "transfer",
		Requires: []over_plugin2.Type{
			over_plugin2.TransferPlugin,
			over_plugin2.StreamingPlugin,
		},
		InitFn: newService,
	})
}

type service struct {
	transferrers  []transfer.Transferrer
	streamManager streaming.StreamManager
	transferapi.UnimplementedTransferServer
}

func newService(ic *over_plugin2.InitContext) (interface{}, error) {
	plugins, err := ic.GetByType(over_plugin2.TransferPlugin)
	if err != nil {
		return nil, err
	}

	// TODO: how to determine order?
	t := make([]transfer.Transferrer, 0, len(plugins))
	for _, p := range plugins {
		i, err := p.Instance()
		if err != nil {
			return nil, err
		}
		t = append(t, i.(transfer.Transferrer))
	}
	sp, err := ic.GetByID(over_plugin2.StreamingPlugin, "manager")
	if err != nil {
		return nil, err
	}
	return &service{
		transferrers:  t,
		streamManager: sp.(streaming.StreamManager),
	}, nil
}

func (s *service) Register(gs *grpc.Server) error {
	transferapi.RegisterTransferServer(gs, s)
	return nil
}

func (s *service) Transfer(ctx context.Context, req *transferapi.TransferRequest) (*emptypb.Empty, error) {
	var transferOpts []transfer.Opt
	if req.Options != nil {
		if req.Options.ProgressStream != "" {
			stream, err := s.streamManager.Get(ctx, req.Options.ProgressStream)
			if err != nil {
				return nil, over_errdefs.ToGRPC(err)
			}
			defer stream.Close()

			pf := func(p transfer.Progress) {
				any, err := typeurl.MarshalAny(&transferTypes.Progress{
					Event:    p.Event,
					Name:     p.Name,
					Parents:  p.Parents,
					Progress: p.Progress,
					Total:    p.Total,
				})
				if err != nil {
					log.G(ctx).WithError(err).Warnf("event could not be marshaled: %v/%v", p.Event, p.Name)
					return
				}
				if err := stream.Send(any); err != nil {
					log.G(ctx).WithError(err).Warnf("event not sent: %v/%v", p.Event, p.Name)
					return
				}
			}

			transferOpts = append(transferOpts, transfer.WithProgress(pf))
		}
	}
	src, err := s.convertAny(ctx, req.Source)
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}
	dst, err := s.convertAny(ctx, req.Destination)
	if err != nil {
		return nil, over_errdefs.ToGRPC(err)
	}

	for _, t := range s.transferrers {
		if err := t.Transfer(ctx, src, dst, transferOpts...); err == nil {
			return &ptypes.Empty{}, nil
		} else if !over_errdefs.IsNotImplemented(err) {
			return nil, over_errdefs.ToGRPC(err)
		}
	}
	return nil, status.Errorf(codes.Unimplemented, "method Transfer not implemented for %s to %s", req.Source.GetTypeUrl(), req.Destination.GetTypeUrl())
}

func (s *service) convertAny(ctx context.Context, a typeurl.Any) (interface{}, error) {
	obj, err := plugins.ResolveType(a)
	if err != nil {
		if over_errdefs.IsNotFound(err) {
			return typeurl.UnmarshalAny(a)
		}
		return nil, err
	}
	switch v := obj.(type) {
	case streamUnmarshaler:
		err = v.UnmarshalAny(ctx, s.streamManager, a)
		return obj, err
	default:
		log.G(ctx).Debug("unmarshling to..")
		err = typeurl.UnmarshalTo(a, obj)
		return obj, err
	}
}

type streamUnmarshaler interface {
	UnmarshalAny(context.Context, streaming.StreamGetter, typeurl.Any) error
}
