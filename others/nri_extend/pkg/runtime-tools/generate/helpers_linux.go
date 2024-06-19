package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moby/sys/mountinfo"
)

func getPropagation(path string) (string, error) {
	var (
		dir = filepath.Clean(path)
		mnt *mountinfo.Info
	)

	mounts, err := mountinfo.GetMounts(mountinfo.ParentsFilter(dir))
	if err != nil {
		return "", err
	}
	if len(mounts) == 0 {
		return "", fmt.Errorf("failed to get mount info for %q", path)
	}

	maxLen := 0
	for _, m := range mounts {
		if l := len(m.Mountpoint); l > maxLen {
			mnt = m
			maxLen = l
		}
	}

	for _, opt := range strings.Split(mnt.Optional, " ") {
		switch {
		case strings.HasPrefix(opt, "shared:"):
			return "rshared", nil
		case strings.HasPrefix(opt, "master:"):
			return "rslave", nil
		}
	}

	return "", nil
}

func ensurePropagation(path string, accepted ...string) error {
	prop, err := getPropagation(path)
	if err != nil {
		return err
	}

	for _, p := range accepted {
		if p == prop {
			return nil
		}
	}

	return fmt.Errorf("path %q mount propagation is %q, not %q",
		path, prop, strings.Join(accepted, " or "))
}
