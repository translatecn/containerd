package gc_scheduler

import "github.com/docker/go-metrics"

var (
	// collectionCounter metrics for counter of gc scheduler collections.
	collectionCounter metrics.LabeledCounter

	// gcTimeHist histogram metrics for duration of gc scheduler collections.
	gcTimeHist metrics.Timer
)

func init() {
	ns := metrics.NewNamespace("containerd", "gc", nil)
	collectionCounter = ns.NewLabeledCounter("collections", "counter of gc scheduler collections", "status")
	gcTimeHist = ns.NewTimer("gc", "duration of gc scheduler collections")
	metrics.Register(ns)
}
