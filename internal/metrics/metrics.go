// Package metrics exposes Prometheus counters/gauges for the control
// plane. It only ever records operational counts (request counts,
// durations, entity counts) — never request/response bodies or secrets.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_requests_total",
		Help: "Total HTTP requests handled by the API.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "api_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	HTTPErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_errors_total",
		Help: "Total HTTP requests that resulted in a 4xx/5xx response.",
	}, []string{"method", "path", "status"})

	ActiveVPNUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "vpn_active_users",
		Help: "Number of VPN users currently ACTIVE.",
	})

	ActiveAccessKeys = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "vpn_active_access_keys",
		Help: "Number of access keys currently ACTIVE.",
	})

	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "vpn_active_connections",
		Help: "Number of active VPN connections across all servers.",
	})

	BandwidthBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vpn_bandwidth_bytes_total",
		Help: "Cumulative VPN bandwidth in bytes.",
	}, []string{"server", "direction"})

	ServerHealthy = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "vpn_server_healthy",
		Help: "1 if the server's Shadowsocks process is reachable, else 0.",
	}, []string{"server"})

	ConfigOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vpn_config_operations_total",
		Help: "Access-key configuration operations (create/revoke/rotate).",
	}, []string{"operation"})
)

// Middleware records request counts, latencies, and error counts per
// route. It uses c.FullPath() (the route template, e.g. "/users/:id")
// rather than the raw URL so per-request identifiers never become a
// Prometheus label value.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
		if c.Writer.Status() >= 400 {
			HTTPErrorsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		}
	}
}
