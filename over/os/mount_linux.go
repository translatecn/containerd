package os

import (
	"demo/over/mount"
	"demo/over/my_mount"
	"golang.org/x/sys/unix"
)

// Mount will call my_mount.Mount to mount the file.
func (RealOS) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return my_mount.Mount(source, target, fstype, flags, data)
}

// Unmount will call Unmount to unmount the file.
func (RealOS) Unmount(target string) error {
	return mount.Unmount(target, unix.MNT_DETACH)
}

// LookupMount gets mount info of a given path.
func (RealOS) LookupMount(path string) (mount.Info, error) {
	return mount.Lookup(path)
}
