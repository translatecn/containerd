package v2

import (
	"context"
	"demo/over/events"
	"demo/over/runtime"
	"github.com/docker/go-metrics"
)

// NewTaskMonitor returns a new cgroups monitor
func NewTaskMonitor(ctx context.Context, publisher events.Publisher, ns *metrics.Namespace) (runtime.TaskMonitor, error) {
	collector := NewCollector(ns)
	return &cgroupsMonitor{
		collector: collector,
		context:   ctx,
		publisher: publisher,
	}, nil
}

type cgroupsMonitor struct {
	collector *Collector
	context   context.Context
	publisher events.Publisher
}

func (m *cgroupsMonitor) Monitor(c runtime.Task, labels map[string]string) error {
	if err := m.collector.Add(c, labels); err != nil {
		return err
	}
	return nil
}

func (m *cgroupsMonitor) Stop(c runtime.Task) error {
	m.collector.Remove(c)
	return nil
}
