package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reverse_proxy_http_requests_total",
		Help: "Total number of HTTP requests proxied.",
	}, []string{"prefix", "code"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "reverse_proxy_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"prefix"})

	tcpConnectionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reverse_proxy_tcp_connections_total",
		Help: "Total number of TCP connections accepted.",
	}, []string{"listen_addr"})

	tcpActiveConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reverse_proxy_tcp_active_connections",
		Help: "Current number of active TCP connections.",
	}, []string{"listen_addr"})
)

func newMetricsServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
}
