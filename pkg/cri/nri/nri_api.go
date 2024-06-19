package nri

import (
	"context"
	criconfig "demo/config/cri"
	"time"

	cri "demo/over/api/cri/v1"
	cstore "demo/pkg/cri/store/container"
	sstore "demo/pkg/cri/store/sandbox"
)

type CRIImplementation interface {
	Config() *criconfig.Config
	SandboxStore() *sstore.Store
	ContainerStore() *cstore.Store
	ContainerMetadataExtensionKey() string
	UpdateContainerResources(context.Context, cstore.Container, *cri.UpdateContainerResourcesRequest, cstore.Status) (cstore.Status, error)
	StopContainer(context.Context, cstore.Container, time.Duration) error
}
