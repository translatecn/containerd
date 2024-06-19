package proxy

import (
	"context"
	"demo/over/log"
	"demo/over/streaming"
	"demo/over/typeurl/v2"
	"errors"
	"io"

	"google.golang.org/protobuf/types/known/anypb"

	transferapi "demo/over/api/services/transfer/v1"
	transfertypes "demo/over/api/types/transfer"
	"demo/over/transfer"
	tstreaming "demo/over/transfer/streaming"
)

type proxyTransferrer struct {
	client        transferapi.TransferClient
	streamCreator streaming.StreamCreator
}

// NewTransferrer returns a new transferrer which communicates over a GRPC
// connection using the containerd transfer API
func NewTransferrer(client transferapi.TransferClient, sc streaming.StreamCreator) transfer.Transferrer {
	return &proxyTransferrer{
		client:        client,
		streamCreator: sc,
	}
}

func (p *proxyTransferrer) Transfer(ctx context.Context, src interface{}, dst interface{}, opts ...transfer.Opt) error {
	o := &transfer.Config{}
	for _, opt := range opts {
		opt(o)
	}
	apiOpts := &transferapi.TransferOptions{}
	if o.Progress != nil {
		sid := tstreaming.GenerateID("progress")
		stream, err := p.streamCreator.Create(ctx, sid)
		if err != nil {
			return err
		}
		apiOpts.ProgressStream = sid
		go func() {
			for {
				a, err := stream.Recv()
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.G(ctx).WithError(err).Error("progress stream failed to recv")
					}
					return
				}
				i, err := typeurl.UnmarshalAny(a)
				if err != nil {
					log.G(ctx).WithError(err).Warnf("failed to unmarshal progress object: %v", a.GetTypeUrl())
				}
				switch v := i.(type) {
				case *transfertypes.Progress:
					o.Progress(transfer.Progress{
						Event:    v.Event,
						Name:     v.Name,
						Parents:  v.Parents,
						Progress: v.Progress,
						Total:    v.Total,
					})
				default:
					log.G(ctx).Warnf("unhandled progress object %T: %v", i, a.GetTypeUrl())
				}
			}
		}()
	}
	asrc, err := p.marshalAny(ctx, src)
	if err != nil {
		return err
	}
	adst, err := p.marshalAny(ctx, dst)
	if err != nil {
		return err
	}
	req := &transferapi.TransferRequest{
		Source: &anypb.Any{
			TypeUrl: asrc.GetTypeUrl(),
			Value:   asrc.GetValue(),
		},
		Destination: &anypb.Any{
			TypeUrl: adst.GetTypeUrl(),
			Value:   adst.GetValue(),
		},
		Options: apiOpts,
	}
	_, err = p.client.Transfer(ctx, req)
	return err
}
func (p *proxyTransferrer) marshalAny(ctx context.Context, i interface{}) (typeurl.Any, error) {
	switch m := i.(type) {
	case streamMarshaler:
		return m.MarshalAny(ctx, p.streamCreator)
	}
	return typeurl.MarshalAny(i)
}

type streamMarshaler interface {
	MarshalAny(context.Context, streaming.StreamCreator) (typeurl.Any, error)
}
