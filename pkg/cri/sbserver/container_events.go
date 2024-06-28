package sbserver

import (
	runtime "demo/pkg/api/cri/v1"
)

func (c *CriService) GetContainerEvents(r *runtime.GetEventsRequest, s runtime.RuntimeService_GetContainerEventsServer) error {
	// TODO (https://github.com/containerd/issues/7318):
	// replace with a real implementation that broadcasts containerEventsChan
	// to all subscribers.
	for event := range c.containerEventsChan {
		if err := s.Send(&event); err != nil {
			return err
		}
	}
	return nil
}
