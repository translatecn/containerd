package overlayutils

import (
	"demo/over/log"
	"demo/over/my_mk"
	"demo/over/userns"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"demo/over/mount"
	kernel "demo/pkg/contrib/seccomp/kernelversion"
)

const (
	// see https://man7.org/linux/man-pages/man2/statfs.2.html
	tmpfsMagic = 0x01021994
)

// SupportsMultipleLowerDir checks if the system supports multiple lowerdirs,
// which is required for the overlay snapshotter. On 4.x kernels, multiple lowerdirs
// are always available (so this check isn't needed), and backported to RHEL and
// CentOS 3.x kernels (3.10.0-693.el7.x86_64 and up). This function is to detect
// support on those kernels, without doing a kernel version compare.
//
// Ported from moby overlay2.

// Supported returns nil when the overlayfs is functional on the system with the root directory.
// Supported is not called during plugin initialization, but exposed for downstream projects which uses
// this snapshotter as a library.

// IsPathOnTmpfs returns whether the path is on a tmpfs or not.
//
// It uses statfs to check if the fs type is TMPFS_MAGIC (0x01021994)
// see https://man7.org/linux/man-pages/man2/statfs.2.html
func IsPathOnTmpfs(d string) bool {
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(d, &stat)
	if err != nil {
		log.L.WithError(err).Warnf("Could not retrieve statfs for %v", d)
		return false
	}

	return stat.Type == tmpfsMagic
}

// NeedsUserXAttr returns whether overlayfs should be mounted with the "userxattr" mount option.
//
// The "userxattr" option is needed for mounting overlayfs inside a user namespace with kernel >= 5.11.
//
// The "userxattr" option is NOT needed for the initial user namespace (aka "the host").
//
// Also, Ubuntu (since circa 2015) and Debian (since 10) with kernel < 5.11 can mount
// the overlayfs in a user namespace without the "userxattr" option.
//
// The corresponding kernel commit: https://github.com/torvalds/linux/commit/2d2f2d7322ff43e0fe92bf8cccdc0b09449bf2e1
// > ovl: user xattr
// >
// > Optionally allow using "user.overlay." namespace instead of "trusted.overlay."
// > ...
// > Disable redirect_dir and metacopy options, because these would allow privilege escalation through direct manipulation of the
// > "user.overlay.redirect" or "user.overlay.metacopy" xattrs.
// > ...
//
// The "userxattr" support is not exposed in "/sys/module/overlay/parameters".
func NeedsUserXAttr(d string) (bool, error) {
	if !userns.RunningInUserNS() {
		// we are the real root (i.e., the root in the initial user NS),
		// so we do never need "userxattr" opt.
		return false, nil
	}

	// userxattr not permitted on tmpfs https://man7.org/linux/man-pages/man5/tmpfs.5.html
	if IsPathOnTmpfs(d) {
		return false, nil
	}

	// Fast path on kernels >= 5.11
	//
	// Keep in mind that distro vendors might be going to backport the patch to older kernels
	// so we can't completely remove the "slow path".
	fiveDotEleven := kernel.KernelVersion{Kernel: 5, Major: 11}
	if ok, err := kernel.GreaterEqualThan(fiveDotEleven); err == nil && ok {
		return true, nil
	}

	tdRoot := filepath.Join(d, "userxattr-check")
	if err := os.RemoveAll(tdRoot); err != nil {
		log.L.WithError(err).Warnf("Failed to remove check directory %v", tdRoot)
	}

	if err := my_mk.MkdirAll(tdRoot, 0700); err != nil {
		return false, err
	}

	defer func() {
		if err := os.RemoveAll(tdRoot); err != nil {
			log.L.WithError(err).Warnf("Failed to remove check directory %v", tdRoot)
		}
	}()

	td, err := my_mk.MkdirTemp(tdRoot, "")
	if err != nil {
		return false, err
	}

	for _, dir := range []string{"lower1", "lower2", "upper", "work", "merged"} {
		if err := my_mk.Mkdir(filepath.Join(td, dir), 0755); err != nil {
			return false, err
		}
	}

	opts := []string{
		"ro",
		fmt.Sprintf("lowerdir=%s:%s,upperdir=%s,workdir=%s", filepath.Join(td, "lower2"), filepath.Join(td, "lower1"), filepath.Join(td, "upper"), filepath.Join(td, "work")),
		"userxattr",
	}

	m := mount.Mount{
		Type:    "overlay",
		Source:  "overlay",
		Options: opts,
	}

	dest := filepath.Join(td, "merged")
	if err := m.Mount(dest); err != nil {
		// Probably the host is running Ubuntu/Debian kernel (< 5.11) with the userns patch but without the userxattr patch.
		// Return false without error.
		log.L.WithError(err).Debugf("cannot mount overlay with \"userxattr\", probably the kernel does not support userxattr")
		return false, nil
	}
	if err := mount.UnmountAll(dest, 0); err != nil {
		log.L.WithError(err).Warnf("Failed to unmount check directory %v", dest)
	}
	return true, nil
}
