package metrics

import (
	"encoding/json"
	"io"
	"log/slog"
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
		http.Error(w, "PieProbeName is empty", http.StatusBadRequest)
		return
	}

	rh.metrics.SetLatencyOnMountProbe(
		receivedData.PieProbeName,
		receivedData.Node,
		receivedData.StorageClass,
		receivedData.ReadLatency,
		receivedData.WriteLatency,
	)
	rh.metrics.IncrementPerformanceOnMountProbeCount(
		receivedData.PieProbeName,
		receivedData.Node,
		receivedData.StorageClass,
		receivedData.PerformanceProbeSucceed,
	)

	if _, err := w.Write([]byte("OK")); err != nil {
		slog.Error("failed to write data", "error", err)
	}
}

func NewReceiver(m MetricsExporter) http.Handler {
	return &receiver{
		metrics: m,
	}
}
