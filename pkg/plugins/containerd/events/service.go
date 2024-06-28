package events

import (
	"context"
	"demo/pkg/plugin"
	"demo/pkg/protobuf"
	ptypes "demo/pkg/protobuf/types"
	"fmt"

	api "demo/pkg/api/services/events/v1"
	apittrpc "demo/pkg/api/services/ttrpc/events/v1"
	"demo/pkg/errdefs"
	"demo/pkg/events"
	"demo/pkg/events/exchange"
	"demo/pkg/ttrpc"
	"google.golang.org/grpc"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "events",
		Requires: []plugin.Type{
			plugin.EventPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			ep, err := ic.GetByID(plugin.EventPlugin, "exchange")
			if err != nil {
				return nil, err
			}
			return NewService(ep.(*exchange.Exchange)), nil
		},
	})
}

type service struct {
	ttService *ttrpcService
	events    *exchange.Exchange
	api.UnimplementedEventsServer
}

// NewService returns the GRPC events server
func NewService(events *exchange.Exchange) api.EventsServer {
	return &service{
		ttService: &ttrpcService{
			events: events,
		},
		events: events,
	}
}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterEventsServer(server, s)
	return nil
}

func (s *service) RegisterTTRPC(server *ttrpc.Server) error {
	apittrpc.RegisterEventsService(server, s.ttService)
	return nil
}

func (s *service) Publish(ctx context.Context, r *api.PublishRequest) (*ptypes.Empty, error) {
	if err := s.events.Publish(ctx, r.Topic, r.Event); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &ptypes.Empty{}, nil
}

func (s *service) Forward(ctx context.Context, r *api.ForwardRequest) (*ptypes.Empty, error) {
	if err := s.events.Forward(ctx, fromProto(r.Envelope)); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &ptypes.Empty{}, nil
}

func (s *service) Subscribe(req *api.SubscribeRequest, srv api.Events_SubscribeServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	eventq, errq := s.events.Subscribe(ctx, req.Filters...)
	for {
		select {
		case ev := <-eventq:
			if err := srv.Send(toProto(ev)); err != nil {
				return fmt.Errorf("failed sending event to subscriber: %w", err)
			}
		case err := <-errq:
			if err != nil {
				return fmt.Errorf("subscription error: %w", err)
			}

			return nil
		}
	}
}

func toProto(env *events.Envelope) *api.Envelope {
	return &api.Envelope{
		Timestamp: protobuf.ToTimestamp(env.Timestamp),
		Namespace: env.Namespace,
		Topic:     env.Topic,
		Event:     protobuf.FromAny(env.Event),
	}
}

func fromProto(env *api.Envelope) *events.Envelope {
	return &events.Envelope{
		Timestamp: protobuf.FromTimestamp(env.Timestamp),
		Namespace: env.Namespace,
		Topic:     env.Topic,
		Event:     env.Event,
	}
}
