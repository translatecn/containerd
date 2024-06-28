package archive

import (
	"archive/tar"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// AufsConvertWhiteout converts whiteout files for aufs.
func AufsConvertWhiteout(_ *tar.Header, _ string) (bool, error) {
	return true, nil
}

// OverlayConvertWhiteout converts whiteout files for overlay.
func OverlayConvertWhiteout(hdr *tar.Header, path string) (bool, error) {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	// if a directory is marked as opaque, we need to translate that to overlay
	if base == whiteoutOpaqueDir {
		// don't write the file itself
		return false, unix.Setxattr(dir, "trusted.overlay.opaque", []byte{'y'}, 0)
	}

	// if a file was deleted and we are using overlay, we need to create a character device
	if strings.HasPrefix(base, whiteoutPrefix) {
		originalBase := base[len(whiteoutPrefix):]
		originalPath := filepath.Join(dir, originalBase)

		if err := unix.Mknod(originalPath, unix.S_IFCHR, 0); err != nil {
			return false, err
		}
		// don't write the file itself
		return false, os.Chown(originalPath, hdr.Uid, hdr.Gid)
	}

	return true, nil
}
