package sbserver

import (
	"context"
	"demo/over/log"
	"fmt"
	"io"

	"demo/containerd"
	runtime "demo/over/api/cri/v1"
	"k8s.io/client-go/tools/remotecommand"

	cio "demo/pkg/cri/io"
)

// Attach prepares a streaming endpoint to attach to a running container, and returns the address.
func (c *CriService) Attach(ctx context.Context, r *runtime.AttachRequest) (*runtime.AttachResponse, error) {
	cntr, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("failed to find container in store: %w", err)
	}
	state := cntr.Status.Get().State()
	if state != runtime.ContainerState_CONTAINER_RUNNING {
		return nil, fmt.Errorf("container is in %s state", criContainerStateToString(state))
	}
	return c.streamServer.GetAttach(r)
}

func (c *CriService) attachContainer(ctx context.Context, id string, stdin io.Reader, stdout, stderr io.WriteCloser,
	tty bool, resize <-chan remotecommand.TerminalSize) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Get container from our container store.
	cntr, err := c.containerStore.Get(id)
	if err != nil {
		return fmt.Errorf("failed to find container %q in store: %w", id, err)
	}
	id = cntr.ID

	state := cntr.Status.Get().State()
	if state != runtime.ContainerState_CONTAINER_RUNNING {
		return fmt.Errorf("container is in %s state", criContainerStateToString(state))
	}

	task, err := cntr.Container.Task(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}
	handleResizing(ctx, resize, func(size remotecommand.TerminalSize) {
		if err := task.Resize(ctx, uint32(size.Width), uint32(size.Height)); err != nil {
			log.G(ctx).WithError(err).Errorf("Failed to resize task %q console", id)
		}
	})

	opts := cio.AttachOptions{
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		Tty:       tty,
		StdinOnce: cntr.Config.StdinOnce,
		CloseStdin: func() error {
			return task.CloseIO(ctx, containerd.WithStdinCloser)
		},
	}
	// TODO(random-liu): Figure out whether we need to support historical output.
	cntr.IO.Attach(opts)
	return nil
}
