package fifo

import (
	"fmt"
	"os"
	"sync"
	"syscall"
)

//nolint:revive
const O_PATH = 010000000

type handle struct {
	f         *os.File
	fd        uintptr
	dev       uint64
	ino       uint64
	closeOnce sync.Once
	name      string
}

func getHandle(fn string) (*handle, error) {
	f, err := os.OpenFile(fn, O_PATH, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v with O_PATH: %w", fn, err)
	}

	var (
		stat syscall.Stat_t
		fd   = f.Fd()
	)
	if err := syscall.Fstat(int(fd), &stat); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to stat handle %v: %w", fd, err)
	}

	h := &handle{
		f:    f,
		name: fn,
		//nolint:unconvert
		dev: stat.Dev,
		ino: stat.Ino,
		fd:  fd,
	}

	// check /proc just in case
	if _, err := os.Stat(h.procPath()); err != nil {
		f.Close()
		return nil, fmt.Errorf("couldn't stat %v: %w", h.procPath(), err)
	}

	return h, nil
}

func (h *handle) procPath() string {
	return fmt.Sprintf("/proc/self/fd/%d", h.fd)
}

func (h *handle) Name() string {
	return h.name
}

func (h *handle) Path() (string, error) {
	var stat syscall.Stat_t
	if err := syscall.Stat(h.procPath(), &stat); err != nil {
		return "", fmt.Errorf("path %v could not be statted: %w", h.procPath(), err)
	}
	//nolint:unconvert
	if uint64(stat.Dev) != h.dev || stat.Ino != h.ino {
		return "", fmt.Errorf("failed to verify handle %v/%v %v/%v", stat.Dev, h.dev, stat.Ino, h.ino)
	}
	return h.procPath(), nil
}

func (h *handle) Close() error {
	h.closeOnce.Do(func() {
		h.f.Close()
	})
	return nil
}
