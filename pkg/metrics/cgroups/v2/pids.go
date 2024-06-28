package v2

import (
	v2 "demo/pkg/metrics/types/v2"
	metrics "github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var pidMetrics = []*metric{
	{
		name: "pids",
		help: "The limit to the number of pids allowed (cgroup v2)",
		unit: metrics.Unit("limit"),
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.Pids == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.Pids.Limit),
				},
			}
		},
	},
	{
		name: "pids",
		help: "The current number of pids (cgroup v2)",
		unit: metrics.Unit("current"),
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.Pids == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.Pids.Current),
				},
			}
		},
	},
}
