package streaming

import (
	"demo/pkg/log"
	"demo/pkg/plugin"
	"demo/pkg/protobuf"
	ptypes "demo/pkg/protobuf/types"
	"demo/pkg/streaming"
	"demo/pkg/typeurl/v2"
	"errors"
	"io"

	api "demo/pkg/api/services/streaming/v1"
	"demo/pkg/errdefs"
	"google.golang.org/grpc"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "streaming",
		Requires: []plugin.Type{
			plugin.StreamingPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			i, err := ic.GetByID(plugin.StreamingPlugin, "manager")
			if err != nil {
				return nil, err
			}
			return &service{manager: i.(streaming.StreamManager)}, nil
		},
	})
}

type service struct {
	manager streaming.StreamManager
	api.UnimplementedStreamingServer
}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterStreamingServer(server, s)
	return nil
}

func (s *service) Stream(srv api.Streaming_StreamServer) error {
	// TODO: Timeout waiting
	a, err := srv.Recv()
	if err != nil {
		return err
	}
	var i api.StreamInit
	if err := typeurl.UnmarshalTo(a, &i); err != nil {
		return err
	}

	// TODO: Save this response to avoid marshaling everytime
	response, err := typeurl.MarshalAny(&ptypes.Empty{})
	if err != nil {
		return err
	}

	cc := make(chan struct{})
	ss := &serviceStream{
		s:  srv,
		cc: cc,
	}

	log.G(srv.Context()).WithField("stream", i.ID).Debug("registering stream")
	if err := s.manager.Register(srv.Context(), i.ID, ss); err != nil {
		return err
	}

	// Send response packet after registering stream
	if err := srv.Send(protobuf.FromAny(response)); err != nil {
		return err
	}

	select {
	case <-srv.Context().Done():
		// TODO: Should return error if not cancelled?
	case <-cc:
	}

	return nil
}

type serviceStream struct {
	s  api.Streaming_StreamServer
	cc chan struct{}
}

func (ss *serviceStream) Send(a typeurl.Any) (err error) {
	err = errdefs.FromGRPC(ss.s.Send(protobuf.FromAny(a)))
	if !errors.Is(err, io.EOF) {
		err = errdefs.FromGRPC(err)
	}
	return
}

func (ss *serviceStream) Recv() (a typeurl.Any, err error) {
	a, err = ss.s.Recv()
	if !errors.Is(err, io.EOF) {
		err = errdefs.FromGRPC(err)
	}
	return
}

func (ss *serviceStream) Close() error {
	select {
	case <-ss.cc:
	default:
		close(ss.cc)
	}
	return nil
}
