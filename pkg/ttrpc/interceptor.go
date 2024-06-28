package ttrpc

import "context"

// UnaryServerInfo provides information about the server request
type UnaryServerInfo struct {
	FullMethod string
}

// UnaryClientInfo provides information about the client request
type UnaryClientInfo struct {
	FullMethod string
}

// StreamServerInfo provides information about the server request
type StreamServerInfo struct {
	FullMethod      string
	StreamingClient bool
	StreamingServer bool
}

// Unmarshaler contains the server request data and allows it to be unmarshaled
// into a concrete type
type Unmarshaler func(interface{}) error

// Invoker invokes the client's request and response from the ttrpc server
type Invoker func(context.Context, *Request, *Response) error

// UnaryServerInterceptor specifies the interceptor function for server request/response
type UnaryServerInterceptor func(context.Context, Unmarshaler, *UnaryServerInfo, Method) (interface{}, error)

// UnaryClientInterceptor specifies the interceptor function for client request/response
type UnaryClientInterceptor func(context.Context, *Request, *Response, *UnaryClientInfo, Invoker) error

func defaultServerInterceptor(ctx context.Context, unmarshal Unmarshaler, _ *UnaryServerInfo, method Method) (interface{}, error) {
	return method(ctx, unmarshal)
}

func defaultClientInterceptor(ctx context.Context, req *Request, resp *Response, _ *UnaryClientInfo, invoker Invoker) error {
	return invoker(ctx, req, resp)
}

type StreamServerInterceptor func(context.Context, StreamServer, *StreamServerInfo, StreamHandler) (interface{}, error)

func defaultStreamServerInterceptor(ctx context.Context, ss StreamServer, _ *StreamServerInfo, stream StreamHandler) (interface{}, error) {
	return stream(ctx, ss)
}

type StreamClientInterceptor func(context.Context)
