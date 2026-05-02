// Package httpmetrics provides reusable Echo middleware that records Prometheus
// metrics for every HTTP request, plus a /metrics endpoint for scraping.
//
// Each service constructs its own Metrics via New, tagging all series with a
// constant "service" label, and exposes /metrics via RegisterMetricsEndpoint.
package httpmetrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// defaultBuckets covers typical sub-second to few-second API latencies. Services
// with slow upstreams (e.g. ai-service calling Ollama) should pass custom
// buckets to New.
var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Metrics holds the request counter and latency histogram for one service.
type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// New builds and registers metrics for the given service. The service name is
// attached as a constant label, so all PromQL queries keep a "service" label.
// Pass nil buckets to use the default latency buckets.
func New(service string, buckets []float64) *Metrics {
	if buckets == nil {
		buckets = defaultBuckets
	}

	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request latency in seconds.",
			ConstLabels: prometheus.Labels{"service": service},
			Buckets:     buckets,
		}, []string{"method", "route"}),
	}

	prometheus.MustRegister(m.requests, m.duration)
	return m
}

// Middleware records request count and latency for every request. The route
// label uses Echo's c.Path() (the route template, e.g. /api/v1/users/:id) so
// path parameters never explode label cardinality.
func (m *Metrics) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Propagate the error; the service's own logger/error handler deals
			// with it. We must not call c.Error here, or it would be handled
			// twice alongside the existing request logger.
			err := next(c)

			route := c.Path()
			if route == "" {
				// Unmatched routes (404s) share a single series instead of one
				// per random URL.
				route = "unmatched"
			}
			method := c.Request().Method
			status := strconv.Itoa(c.Response().Status)

			m.requests.WithLabelValues(method, route, status).Inc()
			m.duration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())

			return err
		}
	}
}

// RegisterMetricsEndpoint exposes GET /metrics for Prometheus to scrape.
func RegisterMetricsEndpoint(e *echo.Echo) {
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
