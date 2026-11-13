package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"
)

func NewServer() *http.Server {
	return &http.Server{Addr: ":8081", Handler: NewHandler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return mux
}
