package sysx

import (
	"bytes"

	"golang.org/x/sys/unix"
)

// Listxattr calls syscall listxattr and reads all content
// and returns a string array
func Listxattr(path string) ([]string, error) {
	return listxattrAll(path, unix.Listxattr)
}

// Setxattr calls syscall setxattr
func Setxattr(path string, attr string, data []byte, flags int) (err error) {
	return unix.Setxattr(path, attr, data, flags)
}

// Getxattr calls syscall getxattr
func Getxattr(path, attr string) ([]byte, error) {
	return getxattrAll(path, attr, unix.Getxattr)
}

// LListxattr lists xattrs, not following symlinks
func LListxattr(path string) ([]string, error) {
	return listxattrAll(path, unix.Llistxattr)
}

// LSetxattr sets an xattr, not following symlinks
func LSetxattr(path string, attr string, data []byte, flags int) (err error) {
	return unix.Lsetxattr(path, attr, data, flags)
}

// LGetxattr gets an xattr, not following symlinks
func LGetxattr(path, attr string) ([]byte, error) {
	return getxattrAll(path, attr, unix.Lgetxattr)
}

const defaultXattrBufferSize = 128

type listxattrFunc func(path string, dest []byte) (int, error)

func listxattrAll(path string, listFunc listxattrFunc) ([]string, error) {
	buf := make([]byte, defaultXattrBufferSize)
	n, err := listFunc(path, buf)
	for err == unix.ERANGE {
		// Buffer too small, use zero-sized buffer to get the actual size
		n, err = listFunc(path, []byte{})
		if err != nil {
			return nil, err
		}
		buf = make([]byte, n)
		n, err = listFunc(path, buf)
	}
	if err != nil {
		return nil, err
	}

	ps := bytes.Split(bytes.TrimSuffix(buf[:n], []byte{0}), []byte{0})
	var entries []string
	for _, p := range ps {
		if len(p) > 0 {
			entries = append(entries, string(p))
		}
	}

	return entries, nil
}

type getxattrFunc func(string, string, []byte) (int, error)

func getxattrAll(path, attr string, getFunc getxattrFunc) ([]byte, error) {
	buf := make([]byte, defaultXattrBufferSize)
	n, err := getFunc(path, attr, buf)
	for err == unix.ERANGE {
		// Buffer too small, use zero-sized buffer to get the actual size
		n, err = getFunc(path, attr, []byte{})
		if err != nil {
			return nil, err
		}
		buf = make([]byte, n)
		n, err = getFunc(path, attr, buf)
	}
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
