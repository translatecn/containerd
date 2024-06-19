package sbserver

import (
	"context"
	criconfig "demo/config/cri"
	"fmt"

	runtime "demo/over/api/cri/v1"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"

	"demo/pkg/cri/opts"
	"demo/pkg/cri/util"
)

// updateOCIResource updates container resource limit.
func updateOCIResource(ctx context.Context, spec *runtimespec.Spec, r *runtime.UpdateContainerResourcesRequest,
	config criconfig.Config) (*runtimespec.Spec, error) {

	// Copy to make sure old spec is not changed.
	var cloned runtimespec.Spec
	if err := util.DeepCopy(&cloned, spec); err != nil {
		return nil, fmt.Errorf("failed to deep copy: %w", err)
	}
	if cloned.Linux == nil {
		cloned.Linux = &runtimespec.Linux{}
	}
	if err := opts.WithResources(r.GetLinux(), config.TolerateMissingHugetlbController, config.DisableHugetlbController)(ctx, nil, nil, &cloned); err != nil {
		return nil, fmt.Errorf("unable to set linux container resources: %w", err)
	}
	return &cloned, nil
}

func getResources(spec *runtimespec.Spec) interface{} {
	return spec.Linux.Resources
}
