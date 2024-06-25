package podsandbox

import (
	"context"
	"demo/over/errdefs"
	"demo/over/protobuf"
	"demo/over/sandbox"
	sandbox2 "demo/pkg/cri/over/store/sandbox"
	ctrdutil "demo/pkg/cri/over/util"
	"fmt"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	eventtypes "demo/over/api/events"
)

func (c *Controller) Stop(ctx context.Context, sandboxID string, _ ...sandbox.StopOpt) error {
	sandbox, err := c.sandboxStore.Get(sandboxID)
	if err != nil {
		return fmt.Errorf("an error occurred when try to find sandbox %q: %w",
			sandboxID, err)
	}

	if err := c.cleanupSandboxFiles(sandboxID, sandbox.Config); err != nil {
		return fmt.Errorf("failed to cleanup sandbox files: %w", err)
	}

	// TODO: The Controller maintains its own Status instead of CRI's sandboxStore.
	// Only stop sandbox container when it's running or unknown.
	state := sandbox.Status.Get().State
	if (state == sandbox2.StateReady || state == sandbox2.StateUnknown) && sandbox.Container != nil {
		if err := c.stopSandboxContainer(ctx, sandbox); err != nil {
			return fmt.Errorf("failed to stop sandbox container %q in %q state: %w", sandboxID, state, err)
		}
	}
	return nil
}

// stopSandboxContainer kills the sandbox container.
// `task.Delete` is not called here because it will be called when
// the event monitor handles the `TaskExit` event.
func (c *Controller) stopSandboxContainer(ctx context.Context, sandbox sandbox2.Sandbox) error {
	id := sandbox.ID
	container := sandbox.Container
	state := sandbox.Status.Get().State
	task, err := container.Task(ctx, nil)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to get sandbox container: %w", err)
		}
		// Don't return for unknown state, some cleanup needs to be done.
		if state == sandbox2.StateUnknown {
			return cleanupUnknownSandbox(ctx, id, sandbox)
		}
		return nil
	}

	// Handle unknown state.
	// The cleanup logic is the same with container unknown state.
	if state == sandbox2.StateUnknown {
		// Start an exit handler for containers in unknown state.
		waitCtx, waitCancel := context.WithCancel(ctrdutil.NamespacedContext())
		defer waitCancel()
		exitCh, err := task.Wait(waitCtx)
		if err != nil {
			if !errdefs.IsNotFound(err) {
				return fmt.Errorf("failed to wait for task: %w", err)
			}
			return cleanupUnknownSandbox(ctx, id, sandbox)
		}

		exitCtx, exitCancel := context.WithCancel(context.Background())
		stopCh := make(chan struct{})
		go func() {
			defer close(stopCh)
			exitStatus, exitedAt, err := c.waitSandboxExit(exitCtx, id, exitCh)
			if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				e := &eventtypes.SandboxExit{
					SandboxID:  id,
					ExitStatus: exitStatus,
					ExitedAt:   protobuf.ToTimestamp(exitedAt),
				}
				logrus.WithError(err).Errorf("Failed to wait sandbox exit %+v", e)
				// TODO: how to backoff
				c.cri.BackOffEvent(id, e)
			}
		}()
		defer func() {
			exitCancel()
			// This ensures that exit monitor is stopped before
			// `Wait` is cancelled, so no exit event is generated
			// because of the `Wait` cancellation.
			<-stopCh
		}()
	}

	// Kill the sandbox container.
	if err = task.Kill(ctx, syscall.SIGKILL); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to kill sandbox container: %w", err)
	}

	return c.waitSandboxStop(ctx, sandbox)
}

// waitSandboxStop waits for sandbox to be stopped until context is cancelled or
// the context deadline is exceeded.
func (c *Controller) waitSandboxStop(ctx context.Context, sandbox sandbox2.Sandbox) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("wait sandbox container %q: %w", sandbox.ID, ctx.Err())
	case <-sandbox.Stopped():
		return nil
	}
}

// cleanupUnknownSandbox cleanup stopped sandbox in unknown state.
func cleanupUnknownSandbox(ctx context.Context, id string, sandbox sandbox2.Sandbox) error {
	// Reuse handleSandboxExit to do the cleanup.
	return handleSandboxExit(ctx, sandbox, &eventtypes.TaskExit{ExitStatus: unknownExitCode, ExitedAt: protobuf.ToTimestamp(time.Now())})
}
