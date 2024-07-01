package fs

import (
	"fmt"
	"os"
	"syscall"
)

// copyIrregular covers devices, pipes, and sockets
func copyIrregular(dst string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t) // not *unix.Stat_t
	if !ok {
		return fmt.Errorf("unsupported stat type: %s: %v", dst, fi.Mode())
	}
	var rDev int
	if fi.Mode()&os.ModeDevice == os.ModeDevice {
		rDev = int(st.Rdev)
	}
	//nolint:unconvert
	return syscall.Mknod(dst, uint32(st.Mode), rDev)
}
