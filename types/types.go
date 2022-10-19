package types

type MetricsExchangeFormat struct {
	Node         string  `json:"node"`
	StorageClass string  `json:"storage_class"`
	WriteLatency float64 `json:"write_latency"`
	ReadLatency  float64 `json:"read_latency"`
}
