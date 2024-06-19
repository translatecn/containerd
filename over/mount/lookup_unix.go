package mount

import (
	"fmt"
	"path/filepath"

	"github.com/moby/sys/mountinfo"
)

// Lookup returns the mount info corresponds to the path.
func Lookup(dir string) (Info, error) {
	dir = filepath.Clean(dir)

	m, err := mountinfo.GetMounts(mountinfo.ParentsFilter(dir))
	if err != nil {
		return Info{}, fmt.Errorf("failed to find the mount info for %q: %w", dir, err)
	}
	if len(m) == 0 {
		return Info{}, fmt.Errorf("failed to find the mount info for %q", dir)
	}

	// find the longest matching mount point
	var idx, maxlen int
	for i := range m {
		if len(m[i].Mountpoint) > maxlen {
			maxlen = len(m[i].Mountpoint)
			idx = i
		}
	}
	return *m[idx], nil
}
