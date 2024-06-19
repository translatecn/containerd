

package blockio

import (
	"demo/over/log"
	"fmt"
	"sync"

	"github.com/intel/goresctrl/pkg/blockio"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

var (
	enabled   bool
	enabledMu sync.RWMutex
)

// IsEnabled checks whether blockio is enabled.
func IsEnabled() bool {
	enabledMu.RLock()
	defer enabledMu.RUnlock()

	return enabled
}

// SetConfig updates blockio config with a given config path.
func SetConfig(configFilePath string) error {
	enabledMu.Lock()
	defer enabledMu.Unlock()

	enabled = false
	if configFilePath == "" {
		log.L.Debug("No blockio config file specified, blockio not configured")
		return nil
	}

	if err := blockio.SetConfigFromFile(configFilePath, true); err != nil {
		return fmt.Errorf("blockio not enabled: %w", err)
	}
	enabled = true
	return nil
}

// ClassNameToLinuxOCI converts blockio class name into the LinuxBlockIO
// structure in the OCI runtime spec.
func ClassNameToLinuxOCI(className string) (*runtimespec.LinuxBlockIO, error) {
	return blockio.OciLinuxBlockIO(className)
}

// ContainerClassFromAnnotations examines container and pod annotations of a
// container and returns its blockio class.
func ContainerClassFromAnnotations(containerName string, containerAnnotations, podAnnotations map[string]string) (string, error) {
	return blockio.ContainerClassFromAnnotations(containerName, containerAnnotations, podAnnotations)
}
