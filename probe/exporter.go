package probe

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/topolvm/csi-driver-availability-monitor/types"
)

type diskInfoImpl struct {
	url          string
	node         string
	storageClass string
}

const (
	maxRetryCount    = 3
	retryIntervalSec = 3
)

func NewDiskInfoExporter(url string, node string, storageClass string) DiskInfoExporter {
	return &diskInfoImpl{
		url:          url,
		node:         node,
		storageClass: storageClass,
	}
}

func (di *diskInfoImpl) Export(metrics DiskMetrics) error {
	m := types.MetricsExchangeFormat{
		Node:         di.node,
		StorageClass: di.storageClass,
		WriteLatency: metrics.WriteLatency,
		ReadLatency:  metrics.ReadLatency,
	}

	s, err := json.Marshal(m)
	if err != nil {
		return err
	}

	for retryCounter := 0; retryCounter < maxRetryCount; retryCounter++ {
		_, err = http.Post(di.url, "application/json", bytes.NewReader(s))
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * retryIntervalSec)
	}

	return err
}
