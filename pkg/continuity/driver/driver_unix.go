package driver

import (
	"demo/pkg/continuity/devices"
	"demo/pkg/continuity/sysx"
	"errors"
	"fmt"
	"os"
	"sort"
)

func (d *driver) Mknod(path string, mode os.FileMode, major, minor int) error {
	err := devices.Mknod(path, mode, major, minor)
	if err != nil {
		err = &os.PathError{Op: "mknod", Path: path, Err: err}
	}
	return err
}

func (d *driver) Mkfifo(path string, mode os.FileMode) error {
	if mode&os.ModeNamedPipe == 0 {
		return errors.New("mode passed to Mkfifo does not have the named pipe bit set")
	}
	// mknod with a mode that has ModeNamedPipe set creates a fifo, not a
	// device.
	err := devices.Mknod(path, mode, 0, 0)
	if err != nil {
		err = &os.PathError{Op: "mkfifo", Path: path, Err: err}
	}
	return err
}

// Getxattr returns all of the extended attributes for the file at path p.
func (d *driver) Getxattr(p string) (map[string][]byte, error) {
	xattrs, err := sysx.Listxattr(p)
	if err != nil {
		return nil, fmt.Errorf("listing %s xattrs: %w", p, err)
	}

	sort.Strings(xattrs)
	m := make(map[string][]byte, len(xattrs))

	for _, attr := range xattrs {
		value, err := sysx.Getxattr(p, attr)
		if err != nil {
			return nil, fmt.Errorf("getting %q xattr on %s: %w", attr, p, err)
		}

		// NOTE(stevvooe): This append/copy tricky relies on unique
		// xattrs. Break this out into an alloc/copy if xattrs are no
		// longer unique.
		m[attr] = append(m[attr], value...)
	}

	return m, nil
}

// Setxattr sets all of the extended attributes on file at path, following
// any symbolic links, if necessary. All attributes on the target are
// replaced by the values from attr. If the operation fails to set any
// attribute, those already applied will not be rolled back.
func (d *driver) Setxattr(path string, attrMap map[string][]byte) error {
	for attr, value := range attrMap {
		if err := sysx.Setxattr(path, attr, value, 0); err != nil {
			return fmt.Errorf("error setting xattr %q on %s: %w", attr, path, err)
		}
	}

	return nil
}

// LGetxattr returns all of the extended attributes for the file at path p
// not following symbolic links.
func (d *driver) LGetxattr(p string) (map[string][]byte, error) {
	xattrs, err := sysx.LListxattr(p)
	if err != nil {
		return nil, fmt.Errorf("listing %s xattrs: %w", p, err)
	}

	sort.Strings(xattrs)
	m := make(map[string][]byte, len(xattrs))

	for _, attr := range xattrs {
		value, err := sysx.LGetxattr(p, attr)
		if err != nil {
			return nil, fmt.Errorf("getting %q xattr on %s: %w", attr, p, err)
		}

		// NOTE(stevvooe): This append/copy tricky relies on unique
		// xattrs. Break this out into an alloc/copy if xattrs are no
		// longer unique.
		m[attr] = append(m[attr], value...)
	}

	return m, nil
}

// LSetxattr sets all of the extended attributes on file at path, not
// following any symbolic links. All attributes on the target are
// replaced by the values from attr. If the operation fails to set any
// attribute, those already applied will not be rolled back.
func (d *driver) LSetxattr(path string, attrMap map[string][]byte) error {
	for attr, value := range attrMap {
		if err := sysx.LSetxattr(path, attr, value, 0); err != nil {
			return fmt.Errorf("error setting xattr %q on %s: %w", attr, path, err)
		}
	}

	return nil
}

func (d *driver) DeviceInfo(fi os.FileInfo) (maj uint64, min uint64, err error) {
	return devices.DeviceInfo(fi)
}