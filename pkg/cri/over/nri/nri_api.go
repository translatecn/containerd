package nri

import (
	"context"
	criconfig "demo/config/cri"
	"demo/pkg/cri/over/store/container"
	sstore "demo/pkg/cri/over/store/sandbox"
	"time"

	cri "demo/over/api/cri/v1"
)

type CRIImplementation interface {
	Config() *criconfig.Config
	SandboxStore() *sstore.Store
	ContainerStore() *container.Store
	ContainerMetadataExtensionKey() string
	UpdateContainerResources(context.Context, container.Container, *cri.UpdateContainerResourcesRequest, container.Status) (container.Status, error)
	StopContainer(context.Context, container.Container, time.Duration) error
}
