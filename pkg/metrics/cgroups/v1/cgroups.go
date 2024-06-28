package v1

import (
	"context"
	cgroups "demo/others/cgroups/v3/cgroup1"
	eventstypes "demo/pkg/api/events"
	"demo/pkg/errdefs"
	"demo/pkg/events"
	"demo/pkg/log"
	"demo/pkg/namespaces"
	"demo/pkg/plugins/containerd/linux"
	"demo/pkg/runtime"
	"errors"
	"github.com/docker/go-metrics"
	"github.com/sirupsen/logrus"
)

// NewTaskMonitor returns a new cgroups monitor
func NewTaskMonitor(ctx context.Context, publisher events.Publisher, ns *metrics.Namespace) (runtime.TaskMonitor, error) {
	collector := NewCollector(ns)
	oom, err := newOOMCollector(ns)
	if err != nil {
		return nil, err
	}
	return &cgroupsMonitor{
		collector: collector,
		oom:       oom,
		context:   ctx,
		publisher: publisher,
	}, nil
}

type cgroupsMonitor struct {
	collector *Collector
	oom       *oomCollector
	context   context.Context
	publisher events.Publisher
}

func (m *cgroupsMonitor) Monitor(c runtime.Task, labels map[string]string) error {
	if err := m.collector.Add(c, labels); err != nil {
		return err
	}
	t, ok := c.(*linux.Task)
	if !ok {
		return nil
	}
	cg, err := t.Cgroup()
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = m.oom.Add(c.ID(), c.Namespace(), cg, m.trigger)
	if errors.Is(err, cgroups.ErrMemoryNotSupported) {
		logrus.WithError(err).Warn("OOM monitoring failed")
		return nil
	}
	return err
}

func (m *cgroupsMonitor) Stop(c runtime.Task) error {
	m.collector.Remove(c)
	return nil
}

func (m *cgroupsMonitor) trigger(id, namespace string, cg cgroups.Cgroup) {
	ctx := namespaces.WithNamespace(m.context, namespace)
	if err := m.publisher.Publish(ctx, runtime.TaskOOMEventTopic, &eventstypes.TaskOOM{
		ContainerID: id,
	}); err != nil {
		log.G(m.context).WithError(err).Error("post OOM event")
	}
}
