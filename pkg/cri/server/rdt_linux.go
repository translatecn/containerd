package server

import (
	"demo/over/rdt"
	"fmt"

	"github.com/sirupsen/logrus"
)

// rdtClassFromAnnotations examines container and pod annotations of a
// container and returns its effective RDT class.
func (c *criService) rdtClassFromAnnotations(containerName string, containerAnnotations, podAnnotations map[string]string) (string, error) {
	cls, err := rdt.ContainerClassFromAnnotations(containerName, containerAnnotations, podAnnotations)

	if err == nil {
		// Our internal check that RDT has been enabled
		if cls != "" && !rdt.IsEnabled() {
			err = fmt.Errorf("RDT disabled, refusing to set RDT class of container %q to %q", containerName, cls)
		}
	}

	if err != nil {
		if !rdt.IsEnabled() && c.config.ContainerdConfig.IgnoreRdtNotEnabledErrors {
			logrus.Debugf("continuing create container %s, ignoring rdt not enabled (%v)", containerName, err)
			return "", nil
		}
		return "", err
	}

	return cls, nil
}
