package sbserver

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"demo/pkg/blockio"
)

func (c *CriService) blockIOClassFromAnnotations(containerName string, containerAnnotations, podAnnotations map[string]string) (string, error) {
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
