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

package streaming

import (
	"demo/others/log"
	"demo/others/typeurl/v2"
	over_plugin2 "demo/over/plugin"
	"demo/over/protobuf"
	ptypes "demo/over/protobuf/types"
	"errors"
	"io"

	"demo/over/errdefs"
	api "demo/pkg/api/services/streaming/v1"
	"demo/pkg/streaming"
	"google.golang.org/grpc"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.GRPCPlugin,
		ID:   "streaming",
		Requires: []over_plugin2.Type{
			over_plugin2.StreamingPlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			i, err := ic.GetByID(over_plugin2.StreamingPlugin, "manager")
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
	if err := srv.Send(over_protobuf.FromAny(response)); err != nil {
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
	err = over_errdefs.FromGRPC(ss.s.Send(over_protobuf.FromAny(a)))
	if !errors.Is(err, io.EOF) {
		err = over_errdefs.FromGRPC(err)
	}
	return
}

func (ss *serviceStream) Recv() (a typeurl.Any, err error) {
	a, err = ss.s.Recv()
	if !errors.Is(err, io.EOF) {
		err = over_errdefs.FromGRPC(err)
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
