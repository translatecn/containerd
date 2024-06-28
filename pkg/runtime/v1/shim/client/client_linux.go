package client

import (
	"demo/others/cgroups/v3/cgroup1"
	"fmt"
	"os/exec"
	"syscall"
)

func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func setCgroup(cgroupPath string, cmd *exec.Cmd) error {
	cg, err := cgroup1.Load(cgroup1.StaticPath(cgroupPath))
	if err != nil {
		return fmt.Errorf("failed to load cgroup %s: %w", cgroupPath, err)
	}
	if err := cg.AddProc(uint64(cmd.Process.Pid)); err != nil {
		return fmt.Errorf("failed to join cgroup %s: %w", cgroupPath, err)
	}
	return nil
}
