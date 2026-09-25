// Command webhook-sink is a tiny demo receiver for the Phase 35 webhook
// module. It exists so the e2e gate and local demos have a hermetic in-cluster
// HTTP target: it records every webhook POST it receives and exposes the last
// N requests at GET /requests (debugging/tests), always accepting with 200.
//
// It is NOT part of the gateway or the service mesh — it is a passive sink
// that the webhooks service delivers to. In compose it runs as its own
// container (`go run ./cmd/webhook-sink`); in the Helm chart as a Deployment
// using the shared backend image binary (/app/bin/webhook-sink).
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"omoikane-backend/internal/observability"
)

// recordedRequest is the shape returned by GET /requests.
type recordedRequest struct {
	Time       time.Time       `json:"time"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	Signature  string          `json:"signature"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage `json:"body"`
}

type sink struct {
	mu       sync.Mutex
	requests []recordedRequest
	maxSeen  int
}

// record appends a received webhook (bounded to the most recent 50 requests).
func (s *sink) record(r recordedRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, r)
	if len(s.requests) > s.maxSeen {
		// Keep the newest maxSeen entries.
		s.requests = s.requests[len(s.requests)-s.maxSeen:]
	}
}

func (s *sink) snapshot() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

func main() {
	// JSON structured logs + process-wide Prometheus registry (Phase 34).
	observability.Setup("webhook-sink")

	port := os.Getenv("SINK_PORT")
	if port == "" {
		port = "8091"
	}

	s := &sink{maxSeen: 50}
	mux := http.NewServeMux()

	mux.Handle("GET /health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	// Prometheus scrape endpoint (Phase 34 consistency).
	mux.Handle("GET /metrics", observability.MetricsHandler())

	mux.Handle("GET /requests", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.snapshot())
	}))

	// Catch-all POST: any path is a webhook delivery into the sink.
	mux.Handle("POST /{path...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		s.record(recordedRequest{
			Time:      time.Now().UTC(),
			Method:    r.Method,
			Path:      r.URL.Path,
			Signature: r.Header.Get("X-Omoikane-Signature"),
			Headers:   map[string]string{"content-type": r.Header.Get("Content-Type"), "user-agent": r.Header.Get("User-Agent")},
			Body:      body,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	addr := ":" + port
	log.Printf("webhook-sink starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("webhook-sink failed: %v", err)
	}
}