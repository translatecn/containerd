package v2

import (
	v2 "demo/pkg/metrics/types/v2"
	metrics "github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// IDName is the name that is used to identify the id being collected in the metric
var IDName = "container_id"

type value struct {
	v float64
	l []string
}

type metric struct {
	name   string
	help   string
	unit   metrics.Unit
	vt     prometheus.ValueType
	labels []string
	// getValues returns the value and labels for the data
	getValues func(stats *v2.Metrics) []value
}

func (m *metric) desc(ns *metrics.Namespace) *prometheus.Desc {
	// the namespace label is for containerd namespaces
	return ns.NewDesc(m.name, m.help, m.unit, append([]string{IDName, "namespace"}, m.labels...)...)
}

func (m *metric) collect(id, namespace string, stats *v2.Metrics, ns *metrics.Namespace, ch chan<- prometheus.Metric, block bool) {
	values := m.getValues(stats)
	for _, v := range values {
		// block signals to block on the sending the metrics so none are missed
		if block {
			ch <- prometheus.MustNewConstMetric(m.desc(ns), m.vt, v.v, append([]string{id, namespace}, v.l...)...)
			continue
		}
		// non-blocking metrics can be dropped if the chan is full
		select {
		case ch <- prometheus.MustNewConstMetric(m.desc(ns), m.vt, v.v, append([]string{id, namespace}, v.l...)...):
		default:
		}
	}
}
