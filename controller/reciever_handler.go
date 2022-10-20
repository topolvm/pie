package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/topolvm/pie/types"
)

type ReceiverHandler struct {
	metrics MetricsExporter
}

func (rh *ReceiverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	bodyString := buf.String()

	receivedData := types.MetricsExchangeFormat{}
	err := json.Unmarshal([]byte(bodyString), &receivedData)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rh.metrics.SetLatency(receivedData.Node, receivedData.StorageClass, receivedData.ReadLatency, receivedData.WriteLatency)

	fmt.Fprintf(w, "OK")
}

func NewReceiverHandler(m MetricsExporter) *ReceiverHandler {
	return &ReceiverHandler{
		metrics: m,
	}
}
