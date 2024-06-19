package server

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"demo/over/blockio"
)

// blockIOClassFromAnnotations examines container and pod annotations of a
// container and returns its effective blockio class.
func (c *criService) blockIOClassFromAnnotations(containerName string, containerAnnotations, podAnnotations map[string]string) (string, error) {
	cls, err := blockio.ContainerClassFromAnnotations(containerName, containerAnnotations, podAnnotations)
	if err != nil {
		return "", err
	}

	if cls != "" && !blockio.IsEnabled() {
		if c.config.ContainerdConfig.IgnoreBlockIONotEnabledErrors {
			cls = ""
			logrus.Debugf("continuing create container %s, ignoring blockio not enabled (%v)", containerName, err)
		} else {
			return "", fmt.Errorf("blockio disabled, refusing to set blockio class of container %q to %q", containerName, cls)
		}
	}
	return cls, nil
}
