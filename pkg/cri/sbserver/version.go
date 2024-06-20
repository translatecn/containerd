package sbserver

import (
	"context"
	"demo/over/version"

	runtime "demo/over/api/cri/v1"

	"demo/pkg/cri/constants"
)

const (
	containerName = "containerd"
	// kubeAPIVersion is the api version of kubernetes.
	// TODO(random-liu): Change this to actual CRI version.
	kubeAPIVersion = "0.1.0"
)

// Version returns the runtime name, runtime version and runtime API version.
func (c *CriService) Version(ctx context.Context, r *runtime.VersionRequest) (*runtime.VersionResponse, error) {
	return &runtime.VersionResponse{
		Version:           kubeAPIVersion,
		RuntimeName:       containerName,
		RuntimeVersion:    version.Version,
		RuntimeApiVersion: constants.CRIVersion,
	}, nil
}
