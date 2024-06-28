package metrics

import (
	"demo/pkg/timeout"
	"demo/pkg/version"
	"time"

	goMetrics "github.com/docker/go-metrics"
)

const (
	ShimStatsRequestTimeout = "io.containerd.timeout.metrics.shimstats"
)

func init() {
	ns := goMetrics.NewNamespace("containerd", "", nil)
	c := ns.NewLabeledCounter("build_info", "containerd build information", "version", "revision")
	c.WithValues(version.Version, version.Revision).Inc()
	goMetrics.Register(ns)
	timeout.Set(ShimStatsRequestTimeout, 2*time.Second)
}
