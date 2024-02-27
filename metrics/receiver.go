package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/topolvm/pie/types"
)

type receiver struct {
	metrics MetricsExporter
}

func (rh *receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var receivedData types.MetricsExchangeFormat
	err = json.Unmarshal(data, &receivedData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if receivedData.PieProbeName == "" {
		rh.metrics.SetLatency(receivedData.Node, receivedData.StorageClass, receivedData.ReadLatency, receivedData.WriteLatency)
		rh.metrics.IncrementPerformanceProbeCount(receivedData.Node, receivedData.StorageClass, receivedData.PerformanceProbeSucceed)
	} else {
		rh.metrics.SetLatencyOnMountProbe(receivedData.PieProbeName, receivedData.Node, receivedData.StorageClass, receivedData.ReadLatency, receivedData.WriteLatency)
		rh.metrics.IncrementPerformanceOnMountProbeCount(receivedData.PieProbeName, receivedData.Node, receivedData.StorageClass, receivedData.PerformanceProbeSucceed)
	}

	fmt.Fprintf(w, "OK")
}

func NewReceiver(m MetricsExporter) http.Handler {
	return &receiver{
		metrics: m,
	}
}
