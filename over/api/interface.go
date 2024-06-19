package api

import (
	"demo/over/ttrpc"
	"google.golang.org/grpc"
)

type GrpcService interface {
	Register(*grpc.Server) error
}

type TtrpcService interface {
	RegisterTTRPC(*ttrpc.Server) error
}
