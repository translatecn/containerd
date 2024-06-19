package archive

import (
	"os"

	"golang.org/x/sys/unix"
)

// mknod wraps Unix.Mknod and casts dev to int
func mknod(path string, mode uint32, dev uint64) error {
	return unix.Mknod(path, mode, int(dev))
}

// lsetxattrCreate wraps unix.Lsetxattr, passes the unix.XATTR_CREATE flag on
// supported operating systems,and ignores appropriate errors
func lsetxattrCreate(link string, attr string, data []byte) error {
	err := unix.Lsetxattr(link, attr, data, unix.XATTR_CREATE)
	if err == unix.ENOTSUP || err == unix.ENODATA || err == unix.EEXIST {
		return nil
	}
	return err
}

// lchmod checks for symlink and changes the mode if not a symlink
func lchmod(path string, mode os.FileMode) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
	}
	return nil
}
