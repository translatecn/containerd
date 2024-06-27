package sbserver

import (
	"context"

	runtime "demo/over/api/cri/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *CriService) ListMetricDescriptors(context.Context, *runtime.ListMetricDescriptorsRequest) (*runtime.ListMetricDescriptorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListMetricDescriptors not implemented")
}
