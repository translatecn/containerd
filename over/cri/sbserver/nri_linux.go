package sbserver

import (
	"context"
	"demo/over/cri/store/container"
	"time"

	cri "demo/over/api/cri/v1"
)

func (i *criImplementation) UpdateContainerResources(ctx context.Context, ctr container.Container, req *cri.UpdateContainerResourcesRequest, status container.Status) (container.Status, error) {
	return i.c.updateContainerResources(ctx, ctr, req, status)
}

func (i *criImplementation) StopContainer(ctx context.Context, ctr container.Container, timeout time.Duration) error {
	return i.c.stopContainer(ctx, ctr, timeout)
}
