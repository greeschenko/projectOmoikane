// Command webhooks is the Omoikane webhook module (Phase 35, flagship demo).
//
// It delivers select platform events (the 7 live backbone types) to external
// HTTP endpoints. Two responsibilities:
//
//  1. Subscriptions CRUD + delivery-log reads (admin). The nginx gateway
//     routes /api/webhooks* here; the admin UI lives at /admin/webhooks.
//  2. Delivery pipeline: a Kafka consumer (group "webhooks") maps backbone
//     CloudEvents into pending WebhookDelivery rows (idempotent via a unique
//     (subscription, event) index), and a pump worker POSTs each payload to
//     the subscription URL — HMAC-SHA256 signed when a secret is configured,
//     retrying with exponential backoff, then expiring (the DLQ-equivalent
//     terminal state) after WEBHOOKS_MAX_ATTEMPTS.
//
// It owns the webhook_* tables in the shared `omoikane` Postgres store
// (process split, shared store — same pattern as settings/messages).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"
	"omoikane-backend/internal/observability"

	"github.com/segmentio/kafka-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// JSON structured logs + process-wide Prometheus registry (Phase 34).
	observability.Setup("webhooks")

	dsn := os.Getenv("WEBHOOKS_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"
	}
	port := os.Getenv("WEBHOOKS_PORT")
	if port == "" {
		port = "8090"
	}

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("webhooks-service: failed to connect to database: %v", err)
	}

	// Migrate the webhook tables (shared omoikane store; the only writer of
	// these tables, so single-writer semantics hold).
	if err := db.AutoMigrate(&models.WebhookSubscription{}, &models.WebhookDelivery{}); err != nil {
		log.Fatalf("webhooks-service: failed to migrate webhook tables: %v", err)
	}
	log.Println("webhooks-service connected and migrated (shared omoikane store)")

	// Migration-only mode (Phase 34): the Helm chart runs each DB service as a
	// post-install migration Job with MIGRATE_ONLY=1 so schema deploys are
	// explicit and restart-safe. Exit cleanly once migrations are applied.
	if os.Getenv("MIGRATE_ONLY") == "1" {
		log.Println("webhooks-service: migrations complete, exiting (MIGRATE_ONLY=1)")
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	// Events wiring: consume the backbone (group "webhooks"). Never replay
	// history into a fresh delivery store — only events published after this
	// group's first start are ingested (committed offsets honored thereafter).
	eventsCfg := events.ConfigFromEnv()
	eventsCfg.ConsumerStartOffset = kafka.LastOffset

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: webhooks-service: ensure kafka topics: %v (consumer will retry)", err)
	}

	consumer, err := events.NewConsumer(eventsCfg, "webhooks", &webhookEventHandler{db: db})
	if err != nil {
		log.Printf("WARNING: webhooks-service: kafka consumer disabled (%v); no events will enqueue deliveries", err)
		consumer = nil
	}

	h := &handlers.Handler{
		DB:        db,
		JWTSecret: jwtSecret,
	}

	mux := newWebhooksMux(h)

	// Delivery pump: HTTP fan-out + retry/backoff/DLQ outside the consumer loop.
	worker := newDeliveryWorker(db)
	applyWorkerEnv(worker)

	addr := ":" + port
	log.Printf("webhooks-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("webhooks-service failed: %v", err)
		}
	}()

	if consumer != nil {
		go runConsumerLoop(ctx, consumer)
	}
	go worker.Run(ctx)

	<-ctx.Done()
	log.Println("webhooks-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("webhooks-service: shutdown error: %v", err)
	}
	if consumer != nil {
		if err := consumer.Close(); err != nil {
			log.Printf("webhooks-service: consumer close error: %v", err)
		}
	}
}

// applyWorkerEnv applies WEBHOOKS_* overrides onto the delivery pump (used to
// tune retry latency in compose/k8s demo vs. production defaults).
func applyWorkerEnv(w *deliveryWorker) {
	if v := os.Getenv("WEBHOOKS_MAX_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			w.maxAttempts = n
		}
	}
	if v := os.Getenv("WEBHOOKS_BACKOFF_CAP_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			w.backoffCap = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("WEBHOOKS_POLL_INTERVAL_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			w.pollInterval = time.Duration(n) * time.Millisecond
		}
	}
}

// newWebhooksMux registers every webhooks-service route. Mux paths carry NO
// /api prefix — nginx prefix locations strip the prefix, so /api/webhooks ->
// /webhooks. Exposed as a function so cmd/webhooks tests exercise the wiring.
func newWebhooksMux(h *handlers.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Prometheus scrape endpoint (Phase 34); service-level, never gateway-exposed.
	mux.Handle("GET /metrics", observability.MetricsHandler())

	// Admin-only webhook surface (gateway-facing, own JWT validation).
	mux.HandleFunc("GET /webhooks", h.Admin(h.GetWebhooks))
	mux.HandleFunc("POST /webhooks", h.Admin(h.CreateWebhook))
	// Literal segment wins over {id} in Go 1.22 ServeMux, so /webhooks/deliveries
	// is reachable next to /webhooks/{id}.
	mux.HandleFunc("GET /webhooks/deliveries", h.Admin(h.GetWebhookDeliveries))
	mux.HandleFunc("GET /webhooks/{id}", h.Admin(h.GetWebhook))
	mux.HandleFunc("PUT /webhooks/{id}", h.Admin(h.UpdateWebhook))
	mux.HandleFunc("DELETE /webhooks/{id}", h.Admin(h.DeleteWebhook))
	mux.HandleFunc("POST /webhooks/{id}/test", h.Admin(h.TestWebhook))

	return mux
}

// runConsumerLoop runs the webhooks consumer until ctx is done, retrying with a
// short backoff on transient Kafka errors so the service survives broker
// restarts/rebalances (mirrors cmd/audit).
func runConsumerLoop(ctx context.Context, consumer *events.Consumer) {
	for {
		err := consumer.Run(ctx)
		if err == nil || ctx.Err() != nil {
			log.Println("webhooks-service: kafka consumer stopped")
			return
		}
		log.Printf("WARNING: webhooks-service: kafka consumer error: %v (retrying in 2s)", err)
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
