package server

import (
	criconfig "demo/config/cri"
	cstore "demo/pkg/cri/store/container"
	sstore "demo/pkg/cri/store/sandbox"
)

type criImplementation struct {
	c *criService
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
