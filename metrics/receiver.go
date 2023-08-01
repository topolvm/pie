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

	rh.metrics.SetLatency(receivedData.Node, receivedData.StorageClass, receivedData.ReadLatency, receivedData.WriteLatency)

	fmt.Fprintf(w, "OK")
}

func NewReceiver(m MetricsExporter) http.Handler {
	return &receiver{
		metrics: m,
	}
}
