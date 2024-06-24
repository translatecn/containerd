package v2

import (
	"context"
	"demo/over/oom"
	"demo/over/plugins/shim/shim"
	"demo/over/runtime"
	"fmt"

	cgroupsv2 "demo/others/cgroups/v3/cgroup2"
	eventstypes "demo/over/api/events"
	"github.com/sirupsen/logrus"
)

// New returns an implementation that listens to OOM events
// from a container's cgroups.
func New(publisher shim.Publisher) (oom.Watcher, error) {
	return &watcher{
		itemCh:    make(chan item),
		publisher: publisher,
	}, nil
}

// watcher implementation for handling OOM events from a container's cgroup
type watcher struct {
	itemCh    chan item
	publisher shim.Publisher
}

type item struct {
	id  string
	ev  cgroupsv2.Event
	err error
}

// Close closes the watcher
func (w *watcher) Close() error {
	return nil
}

// Run the loop
func (w *watcher) Run(ctx context.Context) {
	lastOOMMap := make(map[string]uint64) // key: id, value: ev.OOM
	for {
		select {
		case <-ctx.Done():
			w.Close()
			return
		case i := <-w.itemCh:
			if i.err != nil {
				delete(lastOOMMap, i.id)
				continue
			}
			lastOOM := lastOOMMap[i.id]
			if i.ev.OOMKill > lastOOM {
				if err := w.publisher.Publish(ctx, runtime.TaskOOMEventTopic, &eventstypes.TaskOOM{
					ContainerID: i.id,
				}); err != nil {
					logrus.WithError(err).Error("publish OOM event")
				}
			}
			if i.ev.OOMKill > 0 {
				lastOOMMap[i.id] = i.ev.OOMKill
			}
		}
	}
}

// Add cgroups.Cgroup to the epoll monitor
func (w *watcher) Add(id string, cgx interface{}) error {
	cg, ok := cgx.(*cgroupsv2.Manager)
	if !ok {
		return fmt.Errorf("expected *cgroupsv2.Manager, got: %T", cgx)
	}
	// FIXME: cgroupsv2.Manager does not support closing eventCh routine currently.
	// The routine shuts down when an error happens, mostly when the cgroup is deleted.
	eventCh, errCh := cg.EventChan()
	go func() {
		for {
			i := item{id: id}
			select {
			case ev := <-eventCh:
				i.ev = ev
				w.itemCh <- i
			case err := <-errCh:
				// channel is closed when cgroup gets deleted
				if err != nil {
					i.err = err
					w.itemCh <- i
					// we no longer get any event/err when we got an err
					logrus.WithError(err).Warn("error from *cgroupsv2.Manager.EventChan")
				}
				return
			}
		}
	}()
	return nil
}
