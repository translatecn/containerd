package sbserver

import (
	"context"
	runtime "demo/pkg/api/cri/v1"
	containerdio "demo/pkg/cio"
	"demo/pkg/containerd"
	io2 "demo/pkg/cri/io"
	container2 "demo/pkg/cri/store/container"
	sandboxstore "demo/pkg/cri/store/sandbox"
	ctrdutil "demo/pkg/cri/util"
	"demo/pkg/errdefs"
	cioutil "demo/pkg/ioutil"
	"demo/pkg/log"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

// resetContainerStarting resets the container starting state on start failure. So
// that we could remove the container later.
func resetContainerStarting(container container2.Container) error {
	return container.Status.Update(func(status container2.Status) (container2.Status, error) {
		status.Starting = false
		return status, nil
	})
}

// createContainerLoggers creates container loggers and return write closer for stdout and stderr.
func (c *CriService) createContainerLoggers(logPath string, tty bool) (stdout io.WriteCloser, stderr io.WriteCloser, err error) {
	if logPath != "" {
		// Only generate container log when log path is specified.
		f, err := openLogFile(logPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create and open log file: %w", err)
		}
		defer func() {
			if err != nil {
				f.Close()
			}
		}()
		var stdoutCh, stderrCh <-chan struct{}
		wc := cioutil.NewSerialWriteCloser(f)
		stdout, stdoutCh = io2.NewCRILogger(logPath, wc, io2.Stdout, c.config.MaxContainerLogLineSize)
		// Only redirect stderr when there is no tty.
		if !tty {
			stderr, stderrCh = io2.NewCRILogger(logPath, wc, io2.Stderr, c.config.MaxContainerLogLineSize)
		}
		go func() {
			if stdoutCh != nil {
				<-stdoutCh
			}
			if stderrCh != nil {
				<-stderrCh
			}
			logrus.Debugf("Finish redirecting log file %q, closing it", logPath)
			f.Close()
		}()
	} else {
		stdout = io2.NewDiscardLogger()
		stderr = io2.NewDiscardLogger()
	}
	return
}

func setContainerStarting(container container2.Container) error {
	return container.Status.Update(func(status container2.Status) (container2.Status, error) {
		// Return error if container is not in created state.
		if status.State() != runtime.ContainerState_CONTAINER_CREATED {
			return status, fmt.Errorf("container is in %s state", criContainerStateToString(status.State()))
		}
		// Do not start the container when there is a removal in progress.
		if status.Removing {
			return status, errors.New("container is in removing state, can't be started")
		}
		if status.Starting {
			return status, errors.New("container is already in starting state")
		}
		status.Starting = true
		return status, nil
	})
}

func (c *CriService) StartContainer(ctx context.Context, r *runtime.StartContainerRequest) (retRes *runtime.StartContainerResponse, retErr error) {
	start := time.Now()
	cntr, err := c.containerStore.Get(r.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("an error occurred when try to find container %q: %w", r.GetContainerId(), err)
	}

	info, err := cntr.Container.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("get container info: %w", err)
	}

	id := cntr.ID
	meta := cntr.Metadata
	container := cntr.Container
	config := meta.Config

	// Set starting state to prevent other start/remove operations against this container
	// while it's being started.
	if err := setContainerStarting(cntr); err != nil {
		return nil, fmt.Errorf("failed to set starting state for container %q: %w", id, err)
	}
	defer func() {
		if retErr != nil {
			// Set container to exited if fail to start.
			if err := cntr.Status.UpdateSync(func(status container2.Status) (container2.Status, error) {
				status.Pid = 0
				status.FinishedAt = time.Now().UnixNano()
				status.ExitCode = errorStartExitCode
				status.Reason = errorStartReason
				status.Message = retErr.Error()
				return status, nil
			}); err != nil {
				log.G(ctx).WithError(err).Errorf("failed to set start failure state for container %q", id)
			}
		}
		if err := resetContainerStarting(cntr); err != nil {
			log.G(ctx).WithError(err).Errorf("failed to reset starting state for container %q", id)
		}
	}()

	// Get sandbox config from sandbox store.
	sandbox, err := c.sandboxStore.Get(meta.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("sandbox %q not found: %w", meta.SandboxID, err)
	}
	sandboxID := meta.SandboxID
	if sandbox.Status.Get().State != sandboxstore.StateReady {
		return nil, fmt.Errorf("sandbox container %q is not running", sandboxID)
	}

	// Recheck target container validity in Linux namespace options.
	if linux := config.GetLinux(); linux != nil {
		nsOpts := linux.GetSecurityContext().GetNamespaceOptions()
		if nsOpts.GetPid() == runtime.NamespaceMode_TARGET {
			_, err := c.validateTargetContainer(sandboxID, nsOpts.TargetId)
			if err != nil {
				return nil, fmt.Errorf("invalid target container: %w", err)
			}
		}
	}

	ioCreation := func(id string) (_ containerdio.IO, err error) {
		stdoutWC, stderrWC, err := c.createContainerLoggers(meta.LogPath, config.GetTty())
		if err != nil {
			return nil, fmt.Errorf("failed to create container loggers: %w", err)
		}
		cntr.IO.AddOutput("log", stdoutWC, stderrWC)
		cntr.IO.Pipe()
		return cntr.IO, nil
	}

	ociRuntime, err := c.getSandboxRuntime(sandbox.Config, sandbox.Metadata.RuntimeHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox runtime: %w", err)
	}
	taskOpts := []containerd.NewTaskOpts{}
	if ociRuntime.Path != "" {
		taskOpts = append(taskOpts, containerd.WithRuntimePath(ociRuntime.Path))
	}
	task, err := container.NewTask(ctx, ioCreation, taskOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd task: %w", err)
	}
	defer func() {
		if retErr != nil {
			deferCtx, deferCancel := ctrdutil.DeferContext()
			defer deferCancel()
			// It's possible that task is deleted by event monitor.
			if _, err := task.Delete(deferCtx, containerd.WithProcessKill); err != nil && !errdefs.IsNotFound(err) {
				log.G(ctx).WithError(err).Errorf("Failed to delete containerd task %q", id)
			}
		}
	}()

	// wait is a long running background request, no timeout needed.
	exitCh, err := task.Wait(ctrdutil.NamespacedContext())
	if err != nil {
		return nil, fmt.Errorf("failed to wait for containerd task: %w", err)
	}

	defer func() {
		if retErr != nil {
			deferCtx, deferCancel := ctrdutil.DeferContext()
			defer deferCancel()
			err = c.nri.StopContainer(deferCtx, &sandbox, &cntr)
			if err != nil {
				log.G(ctx).WithError(err).Errorf("NRI stop failed for failed container %q", id)
			}
		}
	}()

	err = c.nri.StartContainer(ctx, &sandbox, &cntr)
	if err != nil {
		log.G(ctx).WithError(err).Errorf("NRI container start failed")
		return nil, fmt.Errorf("NRI container start failed: %w", err)
	}

	// Start containerd task.
	if err := task.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start containerd task %q: %w", id, err)
	}

	// Update container start timestamp.
	if err := cntr.Status.UpdateSync(func(status container2.Status) (container2.Status, error) {
		status.Pid = task.Pid()
		status.StartedAt = time.Now().UnixNano()
		return status, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to update container %q state: %w", id, err)
	}

	// It handles the TaskExit event and update container state after this.
	c.eventMonitor.startContainerExitMonitor(context.Background(), id, task.Pid(), exitCh)

	c.generateAndSendContainerEvent(ctx, id, sandboxID, runtime.ContainerEventType_CONTAINER_STARTED_EVENT)

	err = c.nri.PostStartContainer(ctx, &sandbox, &cntr)
	if err != nil {
		log.G(ctx).WithError(err).Errorf("NRI post-start notification failed")
	}

	containerStartTimer.WithValues(info.Runtime.Name).UpdateSince(start)

	return &runtime.StartContainerResponse{}, nil
}
