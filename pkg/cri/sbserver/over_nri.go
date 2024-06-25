package sbserver

import (
	criconfig "demo/config/cri"
	cstore "demo/pkg/cri/over/store/container"
	sstore "demo/pkg/cri/over/store/sandbox"
)

type criImplementation struct {
	c *CriService
}

func (i *criImplementation) Config() *criconfig.Config {
	return &i.c.config
}

func (i *criImplementation) SandboxStore() *sstore.Store {
	return i.c.sandboxStore
}

func (i *criImplementation) ContainerStore() *cstore.Store {
	return i.c.containerStore
}

func (i *criImplementation) ContainerMetadataExtensionKey() string {
	return containerMetadataExtension
}
