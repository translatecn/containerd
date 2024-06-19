package server

import (
	"context"

	runtime "demo/over/api/cri/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *criService) CheckpointContainer(ctx context.Context, r *runtime.CheckpointContainerRequest) (res *runtime.CheckpointContainerResponse, err error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckpointContainer not implemented")
}
