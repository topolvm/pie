package controller

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
	IncrementCreateProbeFastCount(node string, storageClass string)
	IncrementCreateProbeSlowCount(node string, storageClass string)
}

type metricExporterImpl struct {
	writeLatencyGauge    *prometheus.GaugeVec
	readLatencyGauge     *prometheus.GaugeVec
	createProbeFastCount *prometheus.CounterVec
	createProbeSlowCount *prometheus.CounterVec
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

	m.createProbeFastCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pie",
			Name:      "create_probe_fast_total",
			Help:      "The number of attempts that take less time between the creation of the Pod object and the creation of the container than the threshold.",
		},
		[]string{"node", "storage_class"})

	metrics.Registry.MustRegister(m.createProbeFastCount)

	m.createProbeSlowCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pie",
			Name:      "create_probe_slow_total",
			Help:      "The number of attempts that take more time between the creation of the Pod object and the creation of the container than the threshold.",
		},
		[]string{"node", "storage_class"})

	metrics.Registry.MustRegister(m.createProbeSlowCount)
}

func (m *metricExporterImpl) SetLatency(node string, storageClass string, readLatency, writeLatency float64) {
	m.writeLatencyGauge.WithLabelValues(node, storageClass).Set(writeLatency)
	m.readLatencyGauge.WithLabelValues(node, storageClass).Set(readLatency)
}

func (m *metricExporterImpl) IncrementCreateProbeFastCount(node string, storageClass string) {
	m.createProbeFastCount.WithLabelValues(node, storageClass).Inc()
}

func (m *metricExporterImpl) IncrementCreateProbeSlowCount(node string, storageClass string) {
	m.createProbeSlowCount.WithLabelValues(node, storageClass).Inc()
}
