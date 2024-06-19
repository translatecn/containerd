package sbserver

import (
	"context"

	runtime "demo/over/api/cri/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *criService) ListPodSandboxMetrics(context.Context, *runtime.ListPodSandboxMetricsRequest) (*runtime.ListPodSandboxMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPodSandboxMetrics not implemented")
}
