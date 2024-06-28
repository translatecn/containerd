package sbserver

import (
	"context"
	"fmt"

	runtime "demo/pkg/api/cri/v1"
	"demo/pkg/api/services/tasks/v1"
)

// ContainerStats returns stats of the container. If the container does not
// exist, the call returns an error.
func (c *CriService) ContainerStats(ctx context.Context, in *runtime.ContainerStatsRequest) (*runtime.ContainerStatsResponse, error) {
	cntr, err := c.containerStore.Get(in.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("failed to find container: %w", err)
	}
	request := &tasks.MetricsRequest{Filters: []string{"id==" + cntr.ID}}
	resp, err := c.client.TaskService().Metrics(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics for task: %w", err)
	}
	if len(resp.Metrics) != 1 {
		return nil, fmt.Errorf("unexpected metrics response: %+v", resp.Metrics)
	}

	handler, err := c.getMetricsHandler(ctx, cntr.SandboxID)
	if err != nil {
		return nil, err
	}

	cs, err := handler(cntr.Metadata, resp.Metrics[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode container metrics: %w", err)
	}
	return &runtime.ContainerStatsResponse{Stats: cs}, nil
}
