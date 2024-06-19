package net

import (
	"golang.org/x/sys/unix"
)

func newSocketPairCLOEXEC() ([2]int, error) {
	return unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
}
