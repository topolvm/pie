package types

type MetricsExchangeFormat struct {
	PieProbeName            string  `json:"pie_probe_name"`
	Node                    string  `json:"node"`
	StorageClass            string  `json:"storage_class"`
	WriteLatency            float64 `json:"write_latency"`
	ReadLatency             float64 `json:"read_latency"`
	PerformanceProbeSucceed bool    `json:"performance_probe_succeed"`
}
