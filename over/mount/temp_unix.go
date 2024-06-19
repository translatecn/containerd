package mount

import (
	"demo/over/my_mk"
	"github.com/moby/sys/mountinfo"
	"os"
	"path/filepath"
	"sort"
)

// SetTempMountLocation sets the temporary mount location
func SetTempMountLocation(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if err := my_mk.MkdirAll(root, 0700); err != nil {
		return err
	}
	tempMountLocation = root
	return nil
}

// CleanupTempMounts all temp mounts and remove the directories
func CleanupTempMounts(flags int) (warnings []error, err error) {
	mounts, err := mountinfo.GetMounts(mountinfo.PrefixFilter(tempMountLocation))
	if err != nil {
		return nil, err
	}
	// Make the deepest mount be first
	sort.Slice(mounts, func(i, j int) bool {
		return len(mounts[i].Mountpoint) > len(mounts[j].Mountpoint)
	})
	for _, mount := range mounts {
		if err := UnmountAll(mount.Mountpoint, flags); err != nil {
			warnings = append(warnings, err)
			continue
		}
		if err := os.Remove(mount.Mountpoint); err != nil {
			warnings = append(warnings, err)
		}
	}
	return warnings, nil
}
