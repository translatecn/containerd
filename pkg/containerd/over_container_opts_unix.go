package containerd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"demo/pkg/containers"
	"demo/pkg/errdefs"
	"demo/pkg/mount"
	"github.com/opencontainers/image-spec/identity"
)

// WithRemappedSnapshot creates a new snapshot and remaps the uid/gid for the
// filesystem to be used by a container with user namespaces
func WithRemappedSnapshot(id string, i Image, uid, gid uint32) NewContainerOpts {
	return withRemappedSnapshotBase(id, i, uid, gid, false)
}

// WithRemappedSnapshotView is similar to WithRemappedSnapshot but rootfs is mounted as read-only.

func withRemappedSnapshotBase(id string, i Image, uid, gid uint32, readonly bool) NewContainerOpts {
	return func(ctx context.Context, client *Client, c *containers.Container) error {
		diffIDs, err := i.(*image).i.RootFS(ctx, client.ContentStore(), client.platform)
		if err != nil {
			return err
		}

		var (
			parent   = identity.ChainID(diffIDs).String()
			usernsID = fmt.Sprintf("%s-%d-%d", parent, uid, gid)
		)
		c.Snapshotter, err = client.resolveSnapshotterName(ctx, c.Snapshotter)
		if err != nil {
			return err
		}
		snapshotter, err := client.getSnapshotter(ctx, c.Snapshotter)
		if err != nil {
			return err
		}
		if _, err := snapshotter.Stat(ctx, usernsID); err == nil {
			if _, err := snapshotter.Prepare(ctx, id, usernsID); err == nil {
				c.SnapshotKey = id
				c.Image = i.Name()
				return nil
			} else if !errdefs.IsNotFound(err) {
				return err
			}
		}
		mounts, err := snapshotter.Prepare(ctx, usernsID+"-remap", parent)
		if err != nil {
			return err
		}
		if err := remapRootFS(ctx, mounts, uid, gid); err != nil {
			snapshotter.Remove(ctx, usernsID)
			return err
		}
		if err := snapshotter.Commit(ctx, usernsID, usernsID+"-remap"); err != nil {
			return err
		}
		if readonly {
			_, err = snapshotter.View(ctx, id, usernsID)
		} else {
			_, err = snapshotter.Prepare(ctx, id, usernsID)
		}
		if err != nil {
			return err
		}
		c.SnapshotKey = id
		c.Image = i.Name()
		return nil
	}
}

func remapRootFS(ctx context.Context, mounts []mount.Mount, uid, gid uint32) error {
	return mount.WithTempMount(ctx, mounts, func(root string) error {
		return filepath.Walk(root, incrementFS(root, uid, gid))
	})
}

func incrementFS(root string, uidInc, gidInc uint32) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		var (
			stat = info.Sys().(*syscall.Stat_t)
			u, g = int(stat.Uid + uidInc), int(stat.Gid + gidInc)
		)
		// be sure the lchown the path as to not de-reference the symlink to a host file
		return os.Lchown(path, u, g)
	}
}