package shim

import (
	"demo/pkg/ttrpc"
	"golang.org/x/sys/unix"
)

func newServer(opts ...ttrpc.ServerOpt) (*ttrpc.Server, error) {
	opts = append(opts, ttrpc.WithServerHandshaker(ttrpc.UnixSocketRequireSameUser()))
	return ttrpc.NewServer(opts...)
}

func subreaper() error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(1), 0, 0, 0)
}
