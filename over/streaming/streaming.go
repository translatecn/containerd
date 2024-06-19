package streaming

import (
	"context"
	"demo/over/typeurl/v2"
)

type StreamManager interface {
	StreamGetter
	Register(context.Context, string, Stream) error
}

type StreamGetter interface {
	Get(context.Context, string) (Stream, error)
}

type StreamCreator interface {
	Create(context.Context, string) (Stream, error)
}

type Stream interface {
	// Send sends the object on the stream
	Send(typeurl.Any) error

	// Recv receives an object on the stream
	Recv() (typeurl.Any, error)

	// Close closes the stream
	Close() error
}
