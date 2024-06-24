package events

import (
	"context"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"

	api "demo/over/api/services/ttrpc/events/v1"
	"demo/over/errdefs"
	"demo/over/events"
	"demo/over/events/exchange"
)

type ttrpcService struct {
	events *exchange.Exchange
}

func (s *ttrpcService) Forward(ctx context.Context, r *api.ForwardRequest) (*ptypes.Empty, error) {
	if err := s.events.Forward(ctx, fromTProto(r.Envelope)); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &ptypes.Empty{}, nil
}

func fromTProto(env *api.Envelope) *events.Envelope {
	return &events.Envelope{
		Timestamp: protobuf.FromTimestamp(env.Timestamp),
		Namespace: env.Namespace,
		Topic:     env.Topic,
		Event:     env.Event,
	}
}
