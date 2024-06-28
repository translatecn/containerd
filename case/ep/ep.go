package main

import (
	"golang.org/x/sys/unix"
	"io"
	"sync"
)

type File interface {
	io.ReadWriteCloser

	// Fd returns its file descriptor
	Fd() uintptr
	// Name returns its file name
	Name() string
}

// WinSize specifies the window size of the console
type WinSize struct {
	// Height of the console
	Height uint16
	// Width of the console
	Width uint16
	x     uint16
	y     uint16
}

type Console interface {
	File

	// Resize resizes the console to the provided window size
	Resize(WinSize) error
	// ResizeFrom resizes the calling console to the size of the
	// provided console
	ResizeFrom(Console) error
	// SetRaw sets the console in raw mode
	SetRaw() error
	// DisableEcho disables echo on the console
	DisableEcho() error
	// Reset restores the console to its orignal state
	Reset() error
	// Size returns the window size of the console
	Size() (WinSize, error)
}
type EpollConsole struct {
	Console
	readc  *sync.Cond
	writec *sync.Cond
	sysfd  int
	closed bool
}

func main() { // 跑不起来
	maxEvents := 100
	fdMapping := map[int]*EpollConsole{}
	efd, _ := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	go func() {
		events := make([]unix.EpollEvent, maxEvents)
		for {
			n, err := unix.EpollWait(efd, events, -1)
			if err != nil {
				// EINTR: The call was interrupted by a signal handler before either
				// any of the requested events occurred or the timeout expired
				if err == unix.EINTR {
					continue
				}
				break
			}
			for i := 0; i < n; i++ {
				ev := &events[i]
				// the console is ready to be read from
				if ev.Events&(unix.EPOLLIN|unix.EPOLLHUP|unix.EPOLLERR) != 0 {

					if epfile := fdMapping[int(ev.Fd)]; epfile != nil {
						epfile.readc.Signal()
					}
				}
				// the console is ready to be written to
				if ev.Events&(unix.EPOLLOUT|unix.EPOLLHUP|unix.EPOLLERR) != 0 {
					if epfile := fdMapping[int(ev.Fd)]; epfile != nil {
						epfile.readc.Signal()
					}
				}
			}
		}
	}()

	go func() {
		var sysfd int // todo
		unix.SetNonblock(sysfd, true)
		ev := unix.EpollEvent{
			Events: unix.EPOLLIN | unix.EPOLLOUT | unix.EPOLLRDHUP | unix.EPOLLET,
			Fd:     int32(sysfd),
		}
		unix.EpollCtl(efd, unix.EPOLL_CTL_ADD, sysfd, &ev)

	}()

}
