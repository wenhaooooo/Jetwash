package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	LLMLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "LLM request latency in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"model", "status"},
	)

	LLMTokens = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_tokens_total",
			Help: "Total LLM tokens consumed",
		},
		[]string{"model", "type"},
	)

	LLMRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total LLM requests",
		},
		[]string{"model", "status"},
	)
)

func init() {
	prometheus.MustRegister(LLMLatency, LLMTokens, LLMRequests)
}
