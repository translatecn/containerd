package mount

import (
	"context"
	"demo/over/log"
	"demo/over/my_mk"
	"fmt"
	"os"
)

var tempMountLocation = getTempDir()

// WithTempMount mounts the provided mounts to a temp dir, and pass the temp dir to f.
// The mounts are valid during the call to the f.
// Finally we will unmount and remove the temp dir regardless of the result of f.
func WithTempMount(ctx context.Context, mounts []Mount, f func(root string) error) (err error) {
	root, uerr := my_mk.MkdirTemp(tempMountLocation, "containerd-mount")
	if uerr != nil {
		return fmt.Errorf("failed to create temp dir: %w", uerr)
	}
	// We use Remove here instead of RemoveAll.
	// The RemoveAll will delete the temp dir and all children it contains.
	// When the Unmount fails, RemoveAll will incorrectly delete data from
	// the mounted dir. However, if we use Remove, even though we won't
	// successfully delete the temp dir and it may leak, we won't loss data
	// from the mounted dir.
	// For details, please refer to #1868 #1785.
	defer func() {
		if uerr = os.Remove(root); uerr != nil {
			log.G(ctx).WithError(uerr).WithField("dir", root).Error("failed to remove mount temp dir")
		}
	}()

	// We should do defer first, if not we will not do Unmount when only a part of Mounts are failed.
	defer func() {
		if uerr = UnmountMounts(mounts, root, 0); uerr != nil {
			uerr = fmt.Errorf("failed to unmount %s: %w", root, uerr)
			if err == nil {
				err = uerr
			} else {
				err = fmt.Errorf("%s: %w", uerr.Error(), err)
			}
		}
	}()
	if uerr = All(mounts, root); uerr != nil {
		return fmt.Errorf("failed to mount %s: %w", root, uerr)
	}
	if err := f(root); err != nil {
		return fmt.Errorf("mount callback failed on %s: %w", root, err)
	}
	return nil
}

// WithReadonlyTempMount mounts the provided mounts to a temp dir as readonly,
// and pass the temp dir to f. The mounts are valid during the call to the f.
// Finally we will unmount and remove the temp dir regardless of the result of f.
func WithReadonlyTempMount(ctx context.Context, mounts []Mount, f func(root string) error) (err error) {
	return WithTempMount(ctx, readonlyMounts(mounts), f)
}

func getTempDir() string {
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return xdg
	}
	return os.TempDir()
}
