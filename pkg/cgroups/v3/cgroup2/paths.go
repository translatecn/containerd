package cgroup2

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NestedGroupPath will nest the cgroups based on the calling processes cgroup
// placing its child processes inside its own path

// PidGroupPath will return the correct cgroup paths for an existing process running inside a cgroup
// This is commonly used for the Load function to restore an existing container
func PidGroupPath(pid int) (string, error) {
	p := fmt.Sprintf("/proc/%d/cgroup", pid)
	return parseCgroupFile(p)
}

// VerifyGroupPath verifies the format of group path string g.
// The format is same as the third field in /proc/PID/cgroup.
// e.g. "/user.slice/user-1001.slice/session-1.scope"
//
// g must be a "clean" absolute path starts with "/", and must not contain "/sys/fs/cgroup" prefix.
//
// VerifyGroupPath doesn't verify whether g actually exists on the system.
func VerifyGroupPath(g string) error {
	if !strings.HasPrefix(g, "/") {
		return ErrInvalidGroupPath
	}
	if filepath.Clean(g) != g {
		return ErrInvalidGroupPath
	}
	if strings.HasPrefix(g, "/sys/fs/cgroup") {
		return ErrInvalidGroupPath
	}
	return nil
}
