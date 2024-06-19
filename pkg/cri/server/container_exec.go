package server

import (
	"context"
	"fmt"

	runtime "demo/over/api/cri/v1"
)

// Exec prepares a streaming endpoint to execute a command in the container, and returns the address.
func (c *criService) Exec(ctx context.Context, r *runtime.ExecRequest) (*runtime.ExecResponse, error) {
	cntr, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("failed to find container %q in store: %w", r.GetContainerId(), err)
	}
	state := cntr.Status.Get().State()
	if state != runtime.ContainerState_CONTAINER_RUNNING {
		return nil, fmt.Errorf("container is in %s state", criContainerStateToString(state))
	}
	return c.streamServer.GetExec(r)
}
