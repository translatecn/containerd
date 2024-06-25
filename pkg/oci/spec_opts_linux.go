package oci

import (
	"context"
	cap2 "demo/over/cap"

	"demo/over/containers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// WithHostDevices adds all the hosts device nodes to the container's spec
func WithHostDevices(_ context.Context, _ Client, _ *containers.Container, s *Spec) error {
	setLinux(s)

	devs, err := HostDevices()
	if err != nil {
		return err
	}
	s.Linux.Devices = append(s.Linux.Devices, devs...)
	return nil
}

// WithDevices recursively adds devices from the passed in path and associated cgroup rules for that device.
// If devicePath is a dir it traverses the dir to add all devices in that dir.
// If devicePath is not a dir, it attempts to add the single device.
// If containerPath is not set then the device path is used for the container path.
func WithDevices(devicePath, containerPath, permissions string) SpecOpts {
	return func(_ context.Context, _ Client, _ *containers.Container, s *Spec) error {
		devs, err := getDevices(devicePath, containerPath)
		if err != nil {
			return err
		}
		for i := range devs {
			s.Linux.Devices = append(s.Linux.Devices, devs[i])
			s.Linux.Resources.Devices = append(s.Linux.Resources.Devices, specs.LinuxDeviceCgroup{
				Allow:  true,
				Type:   devs[i].Type,
				Major:  &devs[i].Major,
				Minor:  &devs[i].Minor,
				Access: permissions,
			})
		}
		return nil
	}
}

// WithAllCurrentCapabilities propagates the effective capabilities of the caller process to the container process.
// The capability set may differ from WithAllKnownCapabilities when running in a container.
var WithAllCurrentCapabilities = func(ctx context.Context, client Client, c *containers.Container, s *Spec) error {
	caps, err := cap2.Current()
	if err != nil {
		return err
	}
	return WithCapabilities(caps)(ctx, client, c, s)
}

// WithAllKnownCapabilities sets all the known linux capabilities for the container process
var _ = func(ctx context.Context, client Client, c *containers.Container, s *Spec) error {
	caps := cap2.Known()
	return WithCapabilities(caps)(ctx, client, c, s)
}

func escapeAndCombineArgs(args []string) string {
	panic("not supported")
}
