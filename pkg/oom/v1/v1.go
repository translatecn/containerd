package v1

import (
	"context"
	"demo/pkg/oom"
	"demo/pkg/plugins/shim/shim"
	"demo/pkg/runtime"
	"fmt"
	"sync"

	eventstypes "demo/pkg/api/events"
	"demo/pkg/cgroups/v3/cgroup1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func New(publisher shim.Publisher) (oom.Watcher, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &epoller{
		fd:        fd,
		publisher: publisher,
		set:       make(map[uintptr]*item),
	}, nil
}

// epoller implementation for handling OOM events from a container's cgroup
type epoller struct {
	mu sync.Mutex

	fd        int
	publisher shim.Publisher
	set       map[uintptr]*item
}

type item struct {
	id string
	cg cgroup1.Cgroup
}

// Close the epoll fd
func (e *epoller) Close() error {
	return unix.Close(e.fd)
}

// Run the epoll loop
func (e *epoller) Run(ctx context.Context) {
	var events [128]unix.EpollEvent
	for {
		select {
		case <-ctx.Done():
			e.Close()
			return
		default:
			n, err := unix.EpollWait(e.fd, events[:], -1)
			if err != nil {
				if err == unix.EINTR {
					continue
				}
				logrus.WithError(err).Error("cgroups: epoll wait")
			}
			for i := 0; i < n; i++ {
				e.process(ctx, uintptr(events[i].Fd))
			}
		}
	}
}

func (e *epoller) Add(id string, cgx interface{}) error { //  //load, _ := cgroup1.Load(cgroup1.PidPath(pid))  // load.OOMEventFD()
	cg, ok := cgx.(cgroup1.Cgroup)
	if !ok {
		return fmt.Errorf("expected cgroups.Cgroup, got: %T", cgx)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	fd, err := cg.OOMEventFD()
	if err != nil {
		return err
	}
	e.set[fd] = &item{
		id: id,
		cg: cg,
	}
	event := unix.EpollEvent{
		Fd:     int32(fd),
		Events: unix.EPOLLHUP | unix.EPOLLIN | unix.EPOLLERR,
	}
	return unix.EpollCtl(e.fd, unix.EPOLL_CTL_ADD, int(fd), &event)
}

func (e *epoller) process(ctx context.Context, fd uintptr) {
	flush(fd)
	e.mu.Lock()
	i, ok := e.set[fd]
	if !ok {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()
	if i.cg.State() == cgroup1.Deleted {
		e.mu.Lock()
		delete(e.set, fd)
		e.mu.Unlock()
		unix.Close(int(fd))
		return
	}
	if err := e.publisher.Publish(ctx, runtime.TaskOOMEventTopic, &eventstypes.TaskOOM{
		ContainerID: i.id,
	}); err != nil {
		logrus.WithError(err).Error("publish OOM event")
	}
}

func flush(fd uintptr) error {
	var buf [8]byte
	_, err := unix.Read(int(fd), buf[:])
	return err
}
