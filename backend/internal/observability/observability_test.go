package observability

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLogBridge_EmitsJSON verifies stdlib log lines are lifted into JSON
// records carrying the service attribute, with WARNING: lines mapped to warn.
func TestLogBridge_EmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})).With("service", "auth")
	b := &logBridge{logger: logger}

	n, err := b.Write([]byte("WARNING: kafka down (relay will retry)\n"))
	if err != nil {
		t.Fatalf("bridge write: %v", err)
	}
	if n != len("WARNING: kafka down (relay will retry)\n") {
		t.Fatalf("short write: %d", n)
	}
	b.Write([]byte("connected and migrated\n"))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON lines, got %d: %q", len(lines), buf.String())
	}

	var warn map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &warn); err != nil {
		t.Fatalf("warn line not JSON: %v (%q)", err, lines[0])
	}
	if warn["service"] != "auth" || warn["level"] != "WARN" {
		t.Errorf("warn record wrong: %v", warn)
	}
	if msg, _ := warn["msg"].(string); !strings.Contains(msg, "kafka down") {
		t.Errorf("warn msg wrong: %v", warn["msg"])
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &info); err != nil {
		t.Fatalf("info line not JSON: %v (%q)", err, lines[1])
	}
	if info["level"] != "INFO" || info["service"] != "auth" {
		t.Errorf("info record wrong: %v", info)
	}
}

// TestLogBridge_SplitsPartialLines verifies lines not terminated by a newline
// are buffered until the newline arrives (stdlib log always flushes full lines,
// but the bridge should not corrupt partial writes).
func TestLogBridge_SplitsPartialLines(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	b := &logBridge{logger: logger}

	b.Write([]byte("part1 "))
	b.Write([]byte("part2\n"))
	if got := strings.Count(buf.String(), "\n"); got != 1 {
		t.Fatalf("expected 1 complete line, got %d: %q", got, buf.String())
	}
}

// TestMiddleware_RecordsAndServesMetrics verifies the middleware increments the
// per-route counter and that the /metrics handler exposes it as Prometheus
// text with normalized route labels.
func TestMiddleware_RecordsAndServesMetrics(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.Handle("GET /metrics", MetricsHandler())

	srv := httptest.NewServer(Middleware(mux))
	defer srv.Close()

	for i := 0; i < 3; i++ {
		resp, err := http.Get(srv.URL + "/health")
		if err != nil {
			t.Fatalf("GET /health: %v", err)
		}
		resp.Body.Close()
	}
	resp, err := http.Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatalf("GET /users/42: %v", err)
	}
	resp.Body.Close()

	metricsResp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer metricsResp.Body.Close()
	body := string(mustReadAll(t, metricsResp.Body))
	// Prometheus text exposition sorts labels alphabetically.
	for _, want := range []string{
		`http_requests_total{method="GET",route="/health",service="unknown",status="200"} 3`,
		`http_requests_total{method="GET",route="/users",service="unknown",status="404"} 1`,
		"http_request_duration_seconds_count",
		"http_in_flight_requests",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

// TestStatusRecorder_SwallowsRepeatedWriteHeader is the Phase 37 regression for
// a real production defect: the recorder used to forward EVERY WriteHeader to
// the wrapped writer, so a handler that wrote a status twice made net/http log
// `http: superfluous response.WriteHeader call` — visible in the auth-service
// JSON logs. Only the first status must reach the client, and the recorded one
// must be that first status (not the repeat).
func TestStatusRecorder_SwallowsRepeatedWriteHeader(t *testing.T) {
	var body bytes.Buffer
	sr := &statusRecorder{ResponseWriter: &countingResponseWriter{buf: &body}}

	sr.WriteHeader(http.StatusCreated)
	sr.WriteHeader(http.StatusInternalServerError) // must be ignored
	sr.Write([]byte("payload"))

	if sr.status != http.StatusCreated {
		t.Errorf("recorded status = %d, want 201 (the effective one)", sr.status)
	}
	if got := body.String(); got != "payload" {
		t.Errorf("body = %q, want %q", got, "payload")
	}
}

// countingResponseWriter is a minimal writer that records how many times
// WriteHeader reached it — the defect was a repeat being forwarded.
type countingResponseWriter struct {
	buf      *bytes.Buffer
	statuses []int
}

func (w *countingResponseWriter) Header() http.Header         { return http.Header{} }
func (w *countingResponseWriter) WriteHeader(code int)        { w.statuses = append(w.statuses, code) }
func (w *countingResponseWriter) Write(b []byte) (int, error) { return w.buf.Write(b) }

// TestStatusRecorder_ImplicitOK — a handler that only writes a body never calls
// WriteHeader; the effective status is 200 and must be recorded as such.
func TestStatusRecorder_ImplicitOK(t *testing.T) {
	var body bytes.Buffer
	sr := &statusRecorder{ResponseWriter: &countingResponseWriter{buf: &body}}

	sr.Write([]byte("implicit"))

	if sr.status != http.StatusOK {
		t.Errorf("recorded status = %d, want 200", sr.status)
	}
	if body.String() != "implicit" {
		t.Errorf("body = %q", body.String())
	}
}

func mustReadAll(t *testing.T, r io.Reader) []byte {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	return b
}

// TestRouteLabel verifies the cardinality-bounding normalization.
func TestRouteLabel(t *testing.T) {
	cases := map[string]string{
		"/":                      "/",
		"":                       "/",
		"/health":                "/health",
		"/users":                 "/users",
		"/users/5":               "/users",
		"/users/5/restore":       "/users",
		"/blog/posts/2":          "/blog/posts",
		"/media/file/a.png":      "/media/file",
		"/internal/trash/page":   "/internal/trash",
		"/internal/trash/page/5": "/internal/trash",
	}
	for in, want := range cases {
		if got := routeLabel(in); got != want {
			t.Errorf("routeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
