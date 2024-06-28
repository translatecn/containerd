package api

import (
	"demo/pkg/ttrpc"
	"google.golang.org/grpc"
)

type GrpcService interface {
	Register(*grpc.Server) error
}

type TtrpcService interface {
	RegisterTTRPC(*ttrpc.Server) error
}
