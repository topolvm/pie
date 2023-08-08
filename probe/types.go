package probe

import "context"

type DiskMetrics struct {
	ReadLatency  float64
	WriteLatency float64
	ErrorNumber  int
}

type DiskMetricsInterface interface {
	GetMetrics(ctx context.Context) (*DiskMetrics, error)
}

type DiskInfoExporter interface {
	Export(metrics *DiskMetrics) error
}
