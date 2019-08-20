package status

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// MetricsEndpoint is the endpoint for Prometheus metrics.
	MetricsEndpoint = "/metrics"
)

// Metrics is the handler for the `/metrics` endpoint.
func Metrics() http.Handler {
	return promhttp.Handler()
}
