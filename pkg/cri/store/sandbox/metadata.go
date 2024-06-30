package sandbox

import (
	"demo/pkg/go-cni"
	"encoding/json"
	"fmt"

	runtime "demo/pkg/api/cri/v1"
)

// NOTE(random-liu):
// 1) Metadata is immutable after created.
// 2) Metadata is checkpointed as containerd container label.

// metadataVersion is current version of sandbox metadata.
const metadataVersion = "v1"

// versionedMetadata is the internal versioned sandbox metadata.
type versionedMetadata struct {
	// Version indicates the version of the versioned sandbox metadata.
	Version string
	// Metadata's type is metadataInternal. If not there will be a recursive call in MarshalJSON.
	Metadata metadataInternal
}

// metadataInternal is for internal use.
type metadataInternal Metadata

// Metadata is the unversioned sandbox metadata.
type Metadata struct {
	// ID is the sandbox id.
	ID string
	// Name is the sandbox name.
	Name string
	// Config is the CRI sandbox config.
	Config *runtime.PodSandboxConfig
	// NetNSPath is the network namespace used by the sandbox.
	NetNSPath string
	// IP of Pod if it is attached to non host network
	IP string
	// AdditionalIPs of the Pod if it is attached to non host network
	AdditionalIPs []string
	// RuntimeHandler is the runtime handler name of the pod.
	RuntimeHandler string // default
	// CNIresult resulting configuration for attached network namespace interfaces
	CNIResult *cni.Result
	// ProcessSelinuxLabel is the SELinux process label for the container
	ProcessSelinuxLabel string
}

// MarshalJSON encodes Metadata into bytes in json format.
func (c *Metadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(&versionedMetadata{
		Version:  metadataVersion,
		Metadata: metadataInternal(*c),
	})
}

// UnmarshalJSON decodes Metadata from bytes.
func (c *Metadata) UnmarshalJSON(data []byte) error {
	versioned := &versionedMetadata{}
	if err := json.Unmarshal(data, versioned); err != nil {
		return err
	}
	// Handle old version after upgrade.
	switch versioned.Version {
	case metadataVersion:
		*c = Metadata(versioned.Metadata)
		return nil
	}
	return fmt.Errorf("unsupported version: %q", versioned.Version)
}
