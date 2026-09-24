// Package observability provides the ops-hardening substrate shared by every
// Omoikane service binary (Phase 34):
//
//   - Structured JSON logs via log/slog (stdlib, zero new deps). Setup(service)
//     installs a JSON handler as the slog default AND redirects the stdlib `log`
//     package through the same pipeline, so every existing log.Printf/Println/
//     Fatalf call site inside a service main emits a JSON record with a
//     `service` attribute — no call-site rewrites needed.
//   - Prometheus HTTP metrics (/metrics endpoint). Handler() serves a registry
//     that Middleware() feeds: http_requests_total{service,method,route,status},
//     http_request_duration_seconds{service,method,route} and
//     http_in_flight_requests{service}. Route labels are pointer-normalized to
//     two segments (/users/5 -> /users) to bound cardinality.
package observability

import (
	"bytes"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Setup configures JSON structured logging for the calling service and boots
// the process-wide Prometheus registry labelled with `service`. Call it as the
// first statement of every service main. Returns the structured logger for
// callers that want slog directly (stdlib `log` keeps working, lifted into
// JSON records underneath).
func Setup(service string) *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).With("service", service)
	slog.SetDefault(logger)
	log.SetFlags(0)
	log.SetOutput(&logBridge{logger: logger})
	bootDefault(service)
	return logger
}

// logBridge lifts stdlib `log` output lines into structured slog records.
// Lines prefixed with "WARNING: " map to the warn level; everything else is
// informational. Multi-line writes are split on newlines (log always ends one
// record per line once flags are cleared by Setup).
type logBridge struct {
	logger *slog.Logger
	mu     sync.Mutex
	buf    []byte
}

func (b *logBridge) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	for {
		i := bytes.IndexByte(b.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSpace(string(b.buf[:i]))
		b.buf = b.buf[i+1:]
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "WARNING: ") {
			b.logger.Warn(strings.TrimPrefix(line, "WARNING: "))
		} else {
			b.logger.Info(line)
		}
	}
	return len(p), nil
}