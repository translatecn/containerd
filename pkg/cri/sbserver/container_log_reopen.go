package sbserver

import (
	"context"
	"errors"
	"fmt"

	runtime "demo/over/api/cri/v1"
)

// ReopenContainerLog asks the cri plugin to reopen the stdout/stderr log file for the container.
// This is often called after the log file has been rotated.
func (c *criService) ReopenContainerLog(ctx context.Context, r *runtime.ReopenContainerLogRequest) (*runtime.ReopenContainerLogResponse, error) {
	container, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("an error occurred when try to find container %q: %w", r.GetContainerId(), err)
	}

	if container.Status.Get().State() != runtime.ContainerState_CONTAINER_RUNNING {
		return nil, errors.New("container is not running")
	}

	// Create new container logger and replace the existing ones.
	stdoutWC, stderrWC, err := c.createContainerLoggers(container.LogPath, container.Config.GetTty())
	if err != nil {
		return nil, err
	}
	oldStdoutWC, oldStderrWC := container.IO.AddOutput("log", stdoutWC, stderrWC)
	if oldStdoutWC != nil {
		oldStdoutWC.Close()
	}
	if oldStderrWC != nil {
		oldStderrWC.Close()
	}
	return &runtime.ReopenContainerLogResponse{}, nil
}
