package observability

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics wires a Prometheus registry to HTTP request instrumentation for a
// single service. Every service binary runs exactly one process, so there is
// exactly one Metrics per process (the package-level default).
type Metrics struct {
	service string

	registry *prometheus.Registry
	httpReqs *prometheus.CounterVec   // http_requests_total{service,method,route,status}
	httpDur  *prometheus.HistogramVec // http_request_duration_seconds{service,method,route}
	inFlight *prometheus.GaugeVec     // http_in_flight_requests{service}
}

// NewMetrics creates an isolated registry instrumented for the given service.
// Exported so service tests can build a throwaway registry; the service mains
// normally use the package-level default via Default()/Setup.
func NewMetrics(service string) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		service: service,
		registry: reg,
		httpReqs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests handled, by service/method/route/status.",
		}, []string{"service", "method", "route", "status"}),
		httpDur: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, by service/method/route.",
			Buckets: prometheus.DefBuckets,
		}, []string{"service", "method", "route"}),
		inFlight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "HTTP requests currently being handled, by service.",
		}, []string{"service"}),
	}
	reg.MustRegister(m.httpReqs, m.httpDur, m.inFlight)
	return m
}

// package-level default: one registry per process, labelled with the service
// name passed to Setup. If Setup was never called (unit tests exercising a
// mux), the label falls back to "unknown".
var (
	defaultOnce    sync.Once
	defaultMetrics *Metrics
)

// bootDefault is called by Setup so the registry is born with the real service
// label in production binaries.
func bootDefault(service string) {
	defaultOnce.Do(func() { defaultMetrics = NewMetrics(service) })
}

// Default returns the process-wide metrics registry.
func Default() *Metrics {
	defaultOnce.Do(func() { defaultMetrics = NewMetrics("unknown") })
	return defaultMetrics
}

// Middleware wraps next, recording every request into the process-wide
// registry. Route labels are normalized to at most two segments so per-route
// cardinality stays bounded (/users/5 -> /users).
func Middleware(next http.Handler) http.Handler {
	return Default().Middleware(next)
}

// MetricsHandler serves the process-wide registry as Prometheus text. Every
// service mux registers it on "GET /metrics".
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(Default().registry, promhttp.HandlerOpts{})
}

// Middleware is the Metrics-scoped variant used by Default().Middleware.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.inFlight.WithLabelValues(m.service).Inc()
		defer m.inFlight.WithLabelValues(m.service).Dec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		route := routeLabel(r.URL.Path)
		m.httpReqs.WithLabelValues(m.service, r.Method, route, strconv.Itoa(rec.status)).Inc()
		m.httpDur.WithLabelValues(m.service, r.Method, route).Observe(time.Since(start).Seconds())
	})
}

// statusRecorder captures the response status code through WriteHeader.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// routeLabel reduces a request path to a bounded, cardinality-safe label.
// It keeps at most two segments (/blog/posts/2 -> /blog/posts) and drops a
// trailing numeric segment so item routes collapse onto their resource group
// (/users/5 -> /users), while collection + subresource pairs stay distinct.
// A path without segments maps to "/".
func routeLabel(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "/"
	}
	segs := strings.Split(trimmed, "/")
	if len(segs) > 2 {
		segs = segs[:2]
	}
	if len(segs) == 2 && isNumericID(segs[1]) {
		segs = segs[:1]
	}
	return "/" + strings.Join(segs, "/")
}

// isNumericID reports whether s is a non-empty all-digit path segment, the
// shape of the {id} placeholders in every REST route.
func isNumericID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}