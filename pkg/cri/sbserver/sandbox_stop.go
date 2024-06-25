package sbserver

import (
	"context"
	"demo/over/log"
	sandbox2 "demo/pkg/cri/over/store/sandbox"
	"errors"
	"fmt"
	"time"

	runtime "demo/over/api/cri/v1"
)

// StopPodSandbox stops the sandbox. If there are any running containers in the
// sandbox, they should be forcibly terminated.
func (c *CriService) StopPodSandbox(ctx context.Context, r *runtime.StopPodSandboxRequest) (*runtime.StopPodSandboxResponse, error) {
	sandbox, err := c.sandboxStore.Get(r.GetPodSandboxId())
	if err != nil {
		return nil, fmt.Errorf("an error occurred when try to find sandbox %q: %w",
			r.GetPodSandboxId(), err)
	}

	if err := c.stopPodSandbox(ctx, sandbox); err != nil {
		return nil, err
	}

	return &runtime.StopPodSandboxResponse{}, nil
}

func (c *CriService) stopPodSandbox(ctx context.Context, sandbox sandbox2.Sandbox) error {
	// Use the full sandbox id.
	id := sandbox.ID

	// Stop all containers inside the sandbox. This terminates the container forcibly,
	// and container may still be created, so production should not rely on this behavior.
	// TODO(random-liu): Introduce a state in sandbox to avoid future container creation.
	stop := time.Now()
	containers := c.containerStore.List()
	for _, container := range containers {
		if container.SandboxID != id {
			continue
		}
		// Forcibly stop the container. Do not use `StopContainer`, because it introduces a race
		// if a container is removed after list.
		if err := c.stopContainer(ctx, container, 0); err != nil {
			return fmt.Errorf("failed to stop container %q: %w", container.ID, err)
		}
	}

	// Only stop sandbox container when it's running or unknown.
	state := sandbox.Status.Get().State
	if state == sandbox2.StateReady || state == sandbox2.StateUnknown {
		// Use sandbox controller to stop sandbox
		controller, err := c.getSandboxController(sandbox.Config, sandbox.RuntimeHandler)
		if err != nil {
			return fmt.Errorf("failed to get sandbox controller: %w", err)
		}

		if err := controller.Stop(ctx, id); err != nil {
			return fmt.Errorf("failed to stop sandbox %q: %w", id, err)
		}
	}

	sandboxRuntimeStopTimer.WithValues(sandbox.RuntimeHandler).UpdateSince(stop)

	err := c.nri.StopPodSandbox(ctx, &sandbox)
	if err != nil {
		log.G(ctx).WithError(err).Errorf("NRI sandbox stop notification failed")
	}

	// Teardown network for sandbox.
	if sandbox.NetNS != nil {
		netStop := time.Now()
		// Use empty netns path if netns is not available. This is defined in:
		// https://demo/others/cni/blob/v0.7.0-alpha1/SPEC.md
		if closed, err := sandbox.NetNS.Closed(); err != nil {
			return fmt.Errorf("failed to check network namespace closed: %w", err)
		} else if closed {
			sandbox.NetNSPath = ""
		}
		if err := c.teardownPodNetwork(ctx, sandbox); err != nil {
			return fmt.Errorf("failed to destroy network for sandbox %q: %w", id, err)
		}
		if err := sandbox.NetNS.Remove(); err != nil {
			return fmt.Errorf("failed to remove network namespace for sandbox %q: %w", id, err)
		}
		sandboxDeleteNetwork.UpdateSince(netStop)
	}

	log.G(ctx).Infof("TearDown network for sandbox %q successfully", id)

	return nil
}

// waitSandboxStop waits for sandbox to be stopped until context is cancelled or
// the context deadline is exceeded.
func (c *CriService) waitSandboxStop(ctx context.Context, sandbox sandbox2.Sandbox) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("wait sandbox container %q: %w", sandbox.ID, ctx.Err())
	case <-sandbox.Stopped():
		return nil
	}
}

// teardownPodNetwork removes the network from the pod
func (c *CriService) teardownPodNetwork(ctx context.Context, sandbox sandbox2.Sandbox) error {
	netPlugin := c.getNetworkPlugin(sandbox.RuntimeHandler)
	if netPlugin == nil {
		return errors.New("cni config not initialized")
	}

	var (
		id     = sandbox.ID
		path   = sandbox.NetNSPath
		config = sandbox.Config
	)
	opts, err := cniNamespaceOpts(id, config)
	if err != nil {
		return fmt.Errorf("get cni namespace options: %w", err)
	}

	netStart := time.Now()
	err = netPlugin.Remove(ctx, id, path, opts...)
	networkPluginOperations.WithValues(networkTearDownOp).Inc()
	networkPluginOperationsLatency.WithValues(networkTearDownOp).UpdateSince(netStart)
	if err != nil {
		networkPluginOperationsErrors.WithValues(networkTearDownOp).Inc()
		return err
	}
	return nil
}
