package sbserver

import (
	"context"
	"fmt"

	runtime "demo/over/api/cri/v1"
)

func (c *criService) PodSandboxStats(
	ctx context.Context,
	r *runtime.PodSandboxStatsRequest,
) (*runtime.PodSandboxStatsResponse, error) {

	sandbox, err := c.sandboxStore.Get(r.GetPodSandboxId())
	if err != nil {
		return nil, fmt.Errorf("an error occurred when trying to find sandbox %s: %w", r.GetPodSandboxId(), err)
	}

	podSandboxStats, err := c.podSandboxStats(ctx, sandbox)
	if err != nil {
		return nil, fmt.Errorf("failed to decode pod sandbox metrics %s: %w", r.GetPodSandboxId(), err)
	}

	return &runtime.PodSandboxStatsResponse{Stats: podSandboxStats}, nil
}
