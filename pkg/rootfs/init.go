package rootfs

import (
	"demo/pkg/mount"
)

var (
	initializers = map[string]initializerFunc{}
)

type initializerFunc func(string) error

// Mounter handles mount and unmount
type Mounter interface {
	Mount(target string, mounts ...mount.Mount) error
	Unmount(target string) error
}

// InitRootFS initializes the snapshot for use as a rootfs
