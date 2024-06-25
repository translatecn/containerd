package io

import "github.com/docker/go-metrics"

var (
	inputEntries  metrics.Counter
	outputEntries metrics.Counter
	inputBytes    metrics.Counter
	outputBytes   metrics.Counter
	splitEntries  metrics.Counter
)

func init() {
	// These CRI metrics record input and output logging volume.
	ns := metrics.NewNamespace("containerd", "cri", nil)

	inputEntries = ns.NewCounter("input_entries", "Number of log entries received")
	outputEntries = ns.NewCounter("output_entries", "Number of log entries successfully written to disk")
	inputBytes = ns.NewCounter("input_bytes", "Size of logs received")
	outputBytes = ns.NewCounter("output_bytes", "Size of logs successfully written to disk")
	splitEntries = ns.NewCounter("split_entries", "Number of extra log entries created by splitting the "+
		"original log entry. This happens when the original log entry exceeds length limit. "+
		"This metric does not count the original log entry.")

	metrics.Register(ns)
}
