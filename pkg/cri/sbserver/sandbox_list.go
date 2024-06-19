package sbserver

import (
	"context"
	"time"

	runtime "demo/over/api/cri/v1"

	sandboxstore "demo/pkg/cri/store/sandbox"
)

// ListPodSandbox returns a list of Sandbox.
func (c *criService) ListPodSandbox(ctx context.Context, r *runtime.ListPodSandboxRequest) (*runtime.ListPodSandboxResponse, error) {
	start := time.Now()
	// List all sandboxes from store.
	sandboxesInStore := c.sandboxStore.List()
	var sandboxes []*runtime.PodSandbox
	for _, sandboxInStore := range sandboxesInStore {
		sandboxes = append(sandboxes, toCRISandbox(
			sandboxInStore.Metadata,
			sandboxInStore.Status.Get(),
		))
	}

	sandboxes = c.filterCRISandboxes(sandboxes, r.GetFilter())

	sandboxListTimer.UpdateSince(start)
	return &runtime.ListPodSandboxResponse{Items: sandboxes}, nil
}

// toCRISandbox converts sandbox metadata into CRI pod sandbox.
func toCRISandbox(meta sandboxstore.Metadata, status sandboxstore.Status) *runtime.PodSandbox {
	// Set sandbox state to NOTREADY by default.
	state := runtime.PodSandboxState_SANDBOX_NOTREADY
	if status.State == sandboxstore.StateReady {
		state = runtime.PodSandboxState_SANDBOX_READY
	}
	return &runtime.PodSandbox{
		Id:             meta.ID,
		Metadata:       meta.Config.GetMetadata(),
		State:          state,
		CreatedAt:      status.CreatedAt.UnixNano(),
		Labels:         meta.Config.GetLabels(),
		Annotations:    meta.Config.GetAnnotations(),
		RuntimeHandler: meta.RuntimeHandler,
	}
}

func (c *criService) normalizePodSandboxFilter(filter *runtime.PodSandboxFilter) {
	if sb, err := c.sandboxStore.Get(filter.GetId()); err == nil {
		filter.Id = sb.ID
	}
}

func (c *criService) normalizePodSandboxStatsFilter(filter *runtime.PodSandboxStatsFilter) {
	if sb, err := c.sandboxStore.Get(filter.GetId()); err == nil {
		filter.Id = sb.ID
	}
}

// filterCRISandboxes filters CRISandboxes.
func (c *criService) filterCRISandboxes(sandboxes []*runtime.PodSandbox, filter *runtime.PodSandboxFilter) []*runtime.PodSandbox {
	if filter == nil {
		return sandboxes
	}

	c.normalizePodSandboxFilter(filter)
	filtered := []*runtime.PodSandbox{}
	for _, s := range sandboxes {
		// Filter by id
		if filter.GetId() != "" && filter.GetId() != s.Id {
			continue
		}
		// Filter by state
		if filter.GetState() != nil && filter.GetState().GetState() != s.State {
			continue
		}
		// Filter by label
		if filter.GetLabelSelector() != nil {
			match := true
			for k, v := range filter.GetLabelSelector() {
				got, ok := s.Labels[k]
				if !ok || got != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, s)
	}

	return filtered
}
