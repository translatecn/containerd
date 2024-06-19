package cgroup2

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// LoadAttachCgroupDeviceFilter installs eBPF device filter program to /sys/fs/cgroup/<foo> directory.
//
// Requires the system to be running in cgroup2 unified-mode with kernel >= 4.15 .
//
// https://github.com/torvalds/linux/commit/ebc614f687369f9df99828572b1d85a7c2de3d92
func LoadAttachCgroupDeviceFilter(insts asm.Instructions, license string, dirFD int) (func() error, error) {
	nilCloser := func() error {
		return nil
	}
	spec := &ebpf.ProgramSpec{
		Type:         ebpf.CGroupDevice,
		Instructions: insts,
		License:      license,
	}
	prog, err := ebpf.NewProgram(spec)
	if err != nil {
		return nilCloser, err
	}
	err = link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  dirFD,
		Program: prog,
		Attach:  ebpf.AttachCGroupDevice,
		Flags:   unix.BPF_F_ALLOW_MULTI,
	})
	if err != nil {
		return nilCloser, fmt.Errorf("failed to call BPF_PROG_ATTACH (BPF_CGROUP_DEVICE, BPF_F_ALLOW_MULTI): %w", err)
	}
	closer := func() error {
		err = link.RawDetachProgram(link.RawDetachProgramOptions{
			Target:  dirFD,
			Program: prog,
			Attach:  ebpf.AttachCGroupDevice,
		})
		if err != nil {
			return fmt.Errorf("failed to call BPF_PROG_DETACH (BPF_CGROUP_DEVICE): %w", err)
		}
		return nil
	}
	return closer, nil
}

func isRWM(cgroupPermissions string) bool {
	r := false
	w := false
	m := false
	for _, rn := range cgroupPermissions {
		switch rn {
		case 'r':
			r = true
		case 'w':
			w = true
		case 'm':
			m = true
		}
	}
	return r && w && m
}

// the logic is from runc
// https://demo/3rd_party/runc/blob/master/libcontainer/cgroups/fs/devices_v2.go#L44
func canSkipEBPFError(devices []specs.LinuxDeviceCgroup) bool {
	for _, dev := range devices {
		if dev.Allow || !isRWM(dev.Access) {
			return false
		}
	}
	return true
}
