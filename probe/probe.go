package probe

import (
	"context"
)

func SubMain(pieProbeName string, node string, measurePath string, storageClass string, serverURI string) error {
	context := context.Background()
	diskMetrics := NewDiskMetrics(measurePath)
	infoExporter := NewDiskInfoExporter(serverURI, pieProbeName, node, storageClass)

	metrics, err := diskMetrics.GetMetrics(context)
	if err != nil {
		return err
	}

	err = infoExporter.Export(metrics)
	if err != nil {
		return err
	}

	return nil
}
