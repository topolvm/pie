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
	SetLatencyOnMountProbe(pieProbeName, node, storageClass string, readLatency, writeLatency float64)
	IncrementPerformanceOnMountProbeCount(pieProbeName string, node string, storageClass string, succeed bool)
	IncrementProvisionProbeCount(pieProbeName string, storageClass string, onTime bool)
	IncrementMountProbeCount(pieProbeName string, node string, storageClass string, onTime bool)
}

type metricExporterImpl struct {
	writeLatencyOnMountProbeGauge *prometheus.GaugeVec
	readLatencyOnMountProbeGauge  *prometheus.GaugeVec
	performanceOnMountProbeCount  *prometheus.CounterVec
	provisionProbeCount           *prometheus.CounterVec
	mountProbeCount               *prometheus.CounterVec
}

func NewMetrics() MetricsExporter {
	m := &metricExporterImpl{}
	m.registerMetrics()
	return m
}

func (m *metricExporterImpl) registerMetrics() {
	m.writeLatencyOnMountProbeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pie",
			Name:      "io_write_latency_on_mount_probe_seconds",
			Help:      "IO latency of write.",
		},
		[]string{"pie_probe_name", "node", "storage_class"})

	metrics.Registry.MustRegister(m.writeLatencyOnMountProbeGauge)

	m.readLatencyOnMountProbeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "pie",
			Name:      "io_read_latency_on_mount_probe_seconds",
			Help:      "IO latency of read.",
		},
		[]string{"pie_probe_name", "node", "storage_class"})

	metrics.Registry.MustRegister(m.readLatencyOnMountProbeGauge)

	m.performanceOnMountProbeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pie",
			Name:      "performance_on_mount_probe_total",
			Help:      "The number of performance tests on a probe container.",
		},
		[]string{"pie_probe_name", "node", "storage_class", "succeed"})

	metrics.Registry.MustRegister(m.performanceOnMountProbeCount)

	m.provisionProbeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pie",
			Name:      "provision_probe_total",
			Help:      "The number of attempts that the provision of the Pod object and the creation of the container.",
		},
		[]string{"pie_probe_name", "storage_class", "on_time"})

	metrics.Registry.MustRegister(m.provisionProbeCount)

	m.mountProbeCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pie",
			Name:      "mount_probe_total",
			Help:      "The number of attempts that the mount of the Pod object and the creation of the container.",
		},
		[]string{"pie_probe_name", "node", "storage_class", "on_time"})

	metrics.Registry.MustRegister(m.mountProbeCount)
}

func (m *metricExporterImpl) SetLatencyOnMountProbe(pieProbeName string, node string, storageClass string, readLatency, writeLatency float64) {
	m.writeLatencyOnMountProbeGauge.WithLabelValues(pieProbeName, node, storageClass).Set(writeLatency)
	m.readLatencyOnMountProbeGauge.WithLabelValues(pieProbeName, node, storageClass).Set(readLatency)
}

func (m *metricExporterImpl) IncrementPerformanceOnMountProbeCount(pieProbeName string, node string, storageClass string, succeed bool) {
	succeedStr := "false"
	if succeed {
		succeedStr = "true"
	}
	m.performanceOnMountProbeCount.WithLabelValues(pieProbeName, node, storageClass, succeedStr).Inc()
}

func (m *metricExporterImpl) IncrementProvisionProbeCount(pieProbeName string, storageClass string, onTime bool) {
	onTimeStr := "false"
	if onTime {
		onTimeStr = "true"
	}
	m.provisionProbeCount.WithLabelValues(pieProbeName, storageClass, onTimeStr).Inc()
}

func (m *metricExporterImpl) IncrementMountProbeCount(pieProbeName string, node string, storageClass string, onTime bool) {
	onTimeStr := "false"
	if onTime {
		onTimeStr = "true"
	}
	m.mountProbeCount.WithLabelValues(pieProbeName, node, storageClass, onTimeStr).Inc()
}
