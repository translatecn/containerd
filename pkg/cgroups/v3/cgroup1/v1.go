package cgroup1

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Default returns all the groups in the default cgroups mountpoint in a single hierarchy
func Default() ([]Subsystem, error) {
	root, err := v1MountPoint()
	if err != nil {
		return nil, err
	}
	subsystems, err := defaults(root)
	if err != nil {
		return nil, err
	}
	var enabled []Subsystem
	for _, s := range pathers(subsystems) {
		// check and remove the default groups that do not exist
		if _, err := os.Lstat(s.Path("/")); err == nil {
			enabled = append(enabled, s)
		}
	}
	return enabled, nil
}

// v1MountPoint returns the mount point where the cgroup
// mountpoints are mounted in a single hierarchy
func v1MountPoint() (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var (
			text      = scanner.Text()
			fields    = strings.Split(text, " ")
			numFields = len(fields)
		)
		if numFields < 10 {
			return "", fmt.Errorf("mountinfo: bad entry %q", text)
		}
		if fields[numFields-3] == "cgroup" {
			return filepath.Dir(fields[4]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", ErrMountPointNotExist
}
