package transfer

import (
	"context"
	transferapi "demo/over/api/services/transfer/v1"
	transferTypes "demo/over/api/types/transfer"
	"demo/over/errdefs"
	"demo/over/log"
	"demo/over/plugin"
	ptypes "demo/over/protobuf/types"
	"demo/over/streaming"
	"demo/over/transfer"
	"demo/over/transfer/plugins"
	"demo/over/typeurl/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "transfer",
		Requires: []plugin.Type{
			plugin.TransferPlugin,
			plugin.StreamingPlugin,
		},
		InitFn: newService,
	})
}

type service struct {
	transferrers  []transfer.Transferrer
	streamManager streaming.StreamManager
	transferapi.UnimplementedTransferServer
}

func newService(ic *plugin.InitContext) (interface{}, error) {
	plugins, err := ic.GetByType(plugin.TransferPlugin)
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
	sp, err := ic.GetByID(plugin.StreamingPlugin, "manager")
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
				return nil, errdefs.ToGRPC(err)
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
		return nil, errdefs.ToGRPC(err)
	}
	dst, err := s.convertAny(ctx, req.Destination)
	if err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	for _, t := range s.transferrers {
		if err := t.Transfer(ctx, src, dst, transferOpts...); err == nil {
			return &ptypes.Empty{}, nil
		} else if !errdefs.IsNotImplemented(err) {
			return nil, errdefs.ToGRPC(err)
		}
	}
	return nil, status.Errorf(codes.Unimplemented, "method Transfer not implemented for %s to %s", req.Source.GetTypeUrl(), req.Destination.GetTypeUrl())
}

func (s *service) convertAny(ctx context.Context, a typeurl.Any) (interface{}, error) {
	obj, err := plugins.ResolveType(a)
	if err != nil {
		if errdefs.IsNotFound(err) {
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
