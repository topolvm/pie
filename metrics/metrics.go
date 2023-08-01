package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	MetricsClientSubsystem    = "metrics_client"
	MetricsClientFailTotalKey = "fail_total"
)

type MetricsExporter interface {
	SetLatency(node, storageClass string, readLatency, writeLatency float64)
	IncrementCreateProbeCount(node string, storageClass string, onTime bool)
}

type metricExporterImpl struct {
	writeLatencyGauge *prometheus.GaugeVec
	readLatencyGauge  *prometheus.GaugeVec
	createProbeCount  *prometheus.CounterVec
}

func NewMetrics() MetricsExporter {
	m := &metricExporterImpl{}
	m.registerMetrics()
	return m
}

func (m *metricExporterImpl) registerMetrics() {
	m.writeLatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pie",
			Name:      "io_write_latency_seconds",
			Help:      "IO latency of write.",
		},
		[]string{"node", "storage_class"})

	metrics.Registry.MustRegister(m.writeLatencyGauge)

	m.readLatencyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pie",
			Name:      "io_read_latency_seconds",
			Help:      "IO latency of read.",
		},
		[]string{"node", "storage_class"})

	metrics.Registry.MustRegister(m.readLatencyGauge)

	m.createProbeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pie",
			Name:      "create_probe_total",
			Help:      "The number of attempts that the creation of the Pod object and the creation of the container.",
		},
		[]string{"node", "storage_class", "on_time"})

	metrics.Registry.MustRegister(m.createProbeCount)
}

func (m *metricExporterImpl) SetLatency(node string, storageClass string, readLatency, writeLatency float64) {
	m.writeLatencyGauge.WithLabelValues(node, storageClass).Set(writeLatency)
	m.readLatencyGauge.WithLabelValues(node, storageClass).Set(readLatency)
}

func (m *metricExporterImpl) IncrementCreateProbeCount(node string, storageClass string, onTime bool) {
	onTimeStr := "false"
	if onTime {
		onTimeStr = "true"
	}
	m.createProbeCount.WithLabelValues(node, storageClass, onTimeStr).Inc()
}
