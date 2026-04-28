package probe

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/topolvm/pie/types"
)

type diskInfoImpl struct {
	url          string
	pieProbeName string
	node         string
	storageClass string
}

const (
	maxRetryCount    = 3
	retryIntervalSec = 3
)

func NewDiskInfoExporter(url string, pieProbeName string, node string, storageClass string) DiskInfoExporter {
	return &diskInfoImpl{
		url:          url,
		pieProbeName: pieProbeName,
		node:         node,
		storageClass: storageClass,
	}
}

func (di *diskInfoImpl) Export(metrics *DiskMetrics) error {
	m := types.MetricsExchangeFormat{
		PieProbeName:            di.pieProbeName,
		Node:                    di.node,
		StorageClass:            di.storageClass,
		WriteLatency:            metrics.WriteLatency,
		ReadLatency:             metrics.ReadLatency,
		PerformanceProbeSucceed: metrics.ErrorNumber == 0,
	}

	s, err := json.Marshal(m)
	if err != nil {
		return err
	}

	var resp *http.Response
	for retryCounter := 0; retryCounter < maxRetryCount; retryCounter++ {
		resp, err = http.Post(di.url, "application/json", bytes.NewReader(s))
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		log.Printf("failed to post data: %v", err)
		time.Sleep(time.Second * retryIntervalSec)
	}

	return err
}
