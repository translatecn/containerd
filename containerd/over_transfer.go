package containerd

import (
	"context"
	"demo/over/protobuf"
	"demo/over/streaming"
	"demo/over/typeurl/v2"
	"errors"
	"io"

	streamingapi "demo/over/api/services/streaming/v1"
	transferapi "demo/over/api/services/transfer/v1"
	"demo/over/errdefs"
	"demo/over/transfer"
	"demo/over/transfer/proxy"
)

func (c *Client) Transfer(ctx context.Context, src interface{}, dest interface{}, opts ...transfer.Opt) error {
	ctx, done, err := c.WithLease(ctx)
	if err != nil {
		return err
	}
	defer done(ctx)

	return proxy.NewTransferrer(transferapi.NewTransferClient(c.conn), c.streamCreator()).Transfer(ctx, src, dest, opts...)
}

func (c *Client) streamCreator() streaming.StreamCreator {
	return &streamCreator{
		client: streamingapi.NewStreamingClient(c.conn),
	}
}

type streamCreator struct {
	client streamingapi.StreamingClient
}

func (sc *streamCreator) Create(ctx context.Context, id string) (streaming.Stream, error) {
	stream, err := sc.client.Stream(ctx)
	if err != nil {
		return nil, err
	}

	a, err := typeurl.MarshalAny(&streamingapi.StreamInit{
		ID: id,
	})
	if err != nil {
		return nil, err
	}
	err = stream.Send(protobuf.FromAny(a))
	if err != nil {
		if !errors.Is(err, io.EOF) {
			err = errdefs.FromGRPC(err)
		}
		return nil, err
	}

	// Receive an ack that stream is init and ready
	if _, err = stream.Recv(); err != nil {
		if !errors.Is(err, io.EOF) {
			err = errdefs.FromGRPC(err)
		}
		return nil, err
	}

	return &clientStream{
		s: stream,
	}, nil
}

type clientStream struct {
	s streamingapi.Streaming_StreamClient
}

func (cs *clientStream) Send(a typeurl.Any) (err error) {
	err = cs.s.Send(protobuf.FromAny(a))
	if !errors.Is(err, io.EOF) {
		err = errdefs.FromGRPC(err)
	}
	return
}

func (cs *clientStream) Recv() (a typeurl.Any, err error) {
	a, err = cs.s.Recv()
	if !errors.Is(err, io.EOF) {
		err = errdefs.FromGRPC(err)
	}
	return
}

func (cs *clientStream) Close() error {
	return cs.s.CloseSend()
}
