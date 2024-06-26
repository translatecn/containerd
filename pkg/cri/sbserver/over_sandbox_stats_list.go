package sbserver

import (
	"context"
	runtime "demo/over/api/cri/v1"
	"demo/over/errdefs"
	"demo/over/log"
	sandbox2 "demo/pkg/cri/over/store/sandbox"
	"fmt"
	"github.com/hashicorp/go-multierror"
)

// ListPodSandboxStats returns stats of all ready sandboxes.
func (c *CriService) ListPodSandboxStats(
	ctx context.Context,
	r *runtime.ListPodSandboxStatsRequest,
) (*runtime.ListPodSandboxStatsResponse, error) {
	sandboxes := c.sandboxesForListPodSandboxStatsRequest(r)

	var errs *multierror.Error
	podSandboxStats := new(runtime.ListPodSandboxStatsResponse)
	for _, sandbox := range sandboxes {
		sandboxStats, err := c.podSandboxStats(ctx, sandbox)
		switch {
		case errdefs.IsUnavailable(err):
			log.G(ctx).WithField("podsandboxid", sandbox.ID).Debugf("failed to get pod sandbox stats, this is likely a transient error: %v", err)
		case err != nil:
			errs = multierror.Append(errs, fmt.Errorf("failed to decode sandbox container metrics for sandbox %q: %w", sandbox.ID, err))
		default:
			podSandboxStats.Stats = append(podSandboxStats.Stats, sandboxStats)
		}
	}

	return podSandboxStats, errs.ErrorOrNil()
}

func (c *CriService) sandboxesForListPodSandboxStatsRequest(r *runtime.ListPodSandboxStatsRequest) []sandbox2.Sandbox {
	sandboxesInStore := c.sandboxStore.List()

	if r.GetFilter() == nil {
		return sandboxesInStore
	}

	c.normalizePodSandboxStatsFilter(r.GetFilter())

	var sandboxes []sandbox2.Sandbox
	for _, sandbox := range sandboxesInStore {
		if r.GetFilter().GetId() != "" && sandbox.ID != r.GetFilter().GetId() {
			continue
		}

		if r.GetFilter().GetLabelSelector() != nil &&
			!matchLabelSelector(r.GetFilter().GetLabelSelector(), sandbox.Config.GetLabels()) {
			continue
		}

		// We can't obtain metrics for sandboxes that aren't in ready state
		if sandbox.Status.Get().State != sandbox2.StateReady {
			continue
		}

		sandboxes = append(sandboxes, sandbox)
	}

	return sandboxes
}
