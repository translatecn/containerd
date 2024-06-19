package sbserver

import (
	"demo/containerd"
	"demo/over/plugin"
)

// taskOpts generates task options for a (sandbox) container.
func (c *criService) taskOpts(runtimeType string) []containerd.NewTaskOpts {
	// TODO(random-liu): Remove this after shim v1 is deprecated.
	var taskOpts []containerd.NewTaskOpts

	// c.config.NoPivot is only supported for RuntimeLinuxV1 = "io.containerd.runtime.v1.linux" legacy linux runtime
	// and is not supported for RuntimeRuncV1 = "io.containerd.runc.v1" or  RuntimeRuncV2 = "io.containerd.runc.v2"
	// for RuncV1/2 no pivot is set under the containerd.runtimes.runc.options config see
	// https://github.com/containerd/blob/v1.3.2/runtime/v2/runc/options/oci.pb.go#L26
	if c.config.NoPivot && runtimeType == plugin.RuntimeLinuxV1 {
		taskOpts = append(taskOpts, containerd.WithNoPivotRoot)
	}

	return taskOpts
}
