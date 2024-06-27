package opts

import (
	"context"

	runtime "demo/over/api/cri/v1"
	"demo/over/containers"
	"demo/over/errdefs"
	"demo/over/oci"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

func WithProcessCommandLineOrArgsForWindows(config *runtime.ContainerConfig, image *imagespec.ImageConfig) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *runtimespec.Spec) (err error) {
		return errdefs.ErrNotImplemented
	}
}
