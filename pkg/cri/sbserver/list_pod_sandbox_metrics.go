package sbserver

import (
	"context"

	runtime "demo/pkg/api/cri/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *CriService) ListPodSandboxMetrics(context.Context, *runtime.ListPodSandboxMetricsRequest) (*runtime.ListPodSandboxMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPodSandboxMetrics not implemented")
}
