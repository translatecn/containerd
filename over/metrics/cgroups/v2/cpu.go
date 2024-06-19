package v2

import (
	v2 "demo/over/metrics/types/v2"
	metrics "github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var cpuMetrics = []*metric{
	{
		name: "cpu_usage_usec",
		help: "Total cpu usage (cgroup v2)",
		unit: metrics.Unit("microseconds"),
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.CPU == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.CPU.UsageUsec),
				},
			}
		},
	},
	{
		name: "cpu_user_usec",
		help: "Current cpu usage in user space (cgroup v2)",
		unit: metrics.Unit("microseconds"),
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.CPU == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.CPU.UserUsec),
				},
			}
		},
	},
	{
		name: "cpu_system_usec",
		help: "Current cpu usage in kernel space (cgroup v2)",
		unit: metrics.Unit("microseconds"),
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.CPU == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.CPU.SystemUsec),
				},
			}
		},
	},
	{
		name: "cpu_nr_periods",
		help: "Current cpu number of periods (only if controller is enabled)",
		unit: metrics.Total,
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.CPU == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.CPU.NrPeriods),
				},
			}
		},
	},
	{
		name: "cpu_nr_throttled",
		help: "Total number of times tasks have been throttled (only if controller is enabled)",
		unit: metrics.Total,
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.CPU == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.CPU.NrThrottled),
				},
			}
		},
	},
	{
		name: "cpu_throttled_usec",
		help: "Total time duration for which tasks have been throttled. (only if controller is enabled)",
		unit: metrics.Unit("microseconds"),
		vt:   prometheus.GaugeValue,
		getValues: func(stats *v2.Metrics) []value {
			if stats.CPU == nil {
				return nil
			}
			return []value{
				{
					v: float64(stats.CPU.ThrottledUsec),
				},
			}
		},
	},
}
