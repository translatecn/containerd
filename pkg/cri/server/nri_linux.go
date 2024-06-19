package server

import (
	"context"
	"time"

	cri "demo/over/api/cri/v1"
	cstore "demo/pkg/cri/store/container"
)

func (i *criImplementation) UpdateContainerResources(ctx context.Context, ctr cstore.Container, req *cri.UpdateContainerResourcesRequest, status cstore.Status) (cstore.Status, error) {
	return i.c.updateContainerResources(ctx, ctr, req, status)
}

func (i *criImplementation) StopContainer(ctx context.Context, ctr cstore.Container, timeout time.Duration) error {
	return i.c.stopContainer(ctx, ctr, timeout)
}
