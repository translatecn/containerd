package v1

import (
	v1 "demo/pkg/metrics/types/v1"
	metrics "github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var hugetlbMetrics = []*metric{
	{
		name:   "hugetlb_usage",
		help:   "The hugetlb usage",
		unit:   metrics.Bytes,
		vt:     prometheus.GaugeValue,
		labels: []string{"page"},
		getValues: func(stats *v1.Metrics) []value {
			if stats.Hugetlb == nil {
				return nil
			}
			var out []value
			for _, v := range stats.Hugetlb {
				out = append(out, value{
					v: float64(v.Usage),
					l: []string{v.Pagesize},
				})
			}
			return out
		},
	},
	{
		name:   "hugetlb_failcnt",
		help:   "The hugetlb failcnt",
		unit:   metrics.Total,
		vt:     prometheus.GaugeValue,
		labels: []string{"page"},
		getValues: func(stats *v1.Metrics) []value {
			if stats.Hugetlb == nil {
				return nil
			}
			var out []value
			for _, v := range stats.Hugetlb {
				out = append(out, value{
					v: float64(v.Failcnt),
					l: []string{v.Pagesize},
				})
			}
			return out
		},
	},
	{
		name:   "hugetlb_max",
		help:   "The hugetlb maximum usage",
		unit:   metrics.Bytes,
		vt:     prometheus.GaugeValue,
		labels: []string{"page"},
		getValues: func(stats *v1.Metrics) []value {
			if stats.Hugetlb == nil {
				return nil
			}
			var out []value
			for _, v := range stats.Hugetlb {
				out = append(out, value{
					v: float64(v.Max),
					l: []string{v.Pagesize},
				})
			}
			return out
		},
	},
}
