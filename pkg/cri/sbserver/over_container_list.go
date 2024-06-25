package sbserver

import (
	"context"
	containerstore "demo/pkg/cri/over/store/container"
	"time"

	runtime "demo/over/api/cri/v1"
)

// ListContainers lists all containers matching the filter.
func (c *CriService) ListContainers(ctx context.Context, r *runtime.ListContainersRequest) (*runtime.ListContainersResponse, error) {
	start := time.Now()
	// List all containers from store.
	containersInStore := c.containerStore.List()

	var containers []*runtime.Container
	for _, container := range containersInStore {
		containers = append(containers, toCRIContainer(container))
	}

	containers = c.filterCRIContainers(containers, r.GetFilter())

	containerListTimer.UpdateSince(start)
	return &runtime.ListContainersResponse{Containers: containers}, nil
}

// toCRIContainer converts internal container object into CRI container.
func toCRIContainer(container containerstore.Container) *runtime.Container {
	status := container.Status.Get()
	return &runtime.Container{
		Id:           container.ID,
		PodSandboxId: container.SandboxID,
		Metadata:     container.Config.GetMetadata(),
		Image:        container.Config.GetImage(),
		ImageRef:     container.ImageRef,
		State:        status.State(),
		CreatedAt:    status.CreatedAt,
		Labels:       container.Config.GetLabels(),
		Annotations:  container.Config.GetAnnotations(),
	}
}

func (c *CriService) normalizeContainerFilter(filter *runtime.ContainerFilter) {
	if cntr, err := c.containerStore.Get(filter.GetId()); err == nil {
		filter.Id = cntr.ID
	}
	if sb, err := c.sandboxStore.Get(filter.GetPodSandboxId()); err == nil {
		filter.PodSandboxId = sb.ID
	}
}

// filterCRIContainers filters CRIContainers.
func (c *CriService) filterCRIContainers(containers []*runtime.Container, filter *runtime.ContainerFilter) []*runtime.Container {
	if filter == nil {
		return containers
	}

	// The containerd cri plugin supports short ids so long as there is only one
	// match. So we do a lookup against the store here if a pod id has been
	// included in the filter.
	sb := filter.GetPodSandboxId()
	if sb != "" {
		sandbox, err := c.sandboxStore.Get(sb)
		if err == nil {
			sb = sandbox.ID
		}
	}

	c.normalizeContainerFilter(filter)
	filtered := []*runtime.Container{}
	for _, cntr := range containers {
		if filter.GetId() != "" && filter.GetId() != cntr.Id {
			continue
		}
		if sb != "" && sb != cntr.PodSandboxId {
			continue
		}
		if filter.GetState() != nil && filter.GetState().GetState() != cntr.State {
			continue
		}
		if filter.GetLabelSelector() != nil {
			match := true
			for k, v := range filter.GetLabelSelector() {
				got, ok := cntr.Labels[k]
				if !ok || got != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, cntr)
	}

	return filtered
}
