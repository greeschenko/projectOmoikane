// Command media is the Omoikane media service (Phase 30, Wave 2).
//
// It owns the media surface that was previously served by the monolith
// (cmd/api): upload, list, update (alt), delete, batch and — critically — the
// CDN-ready public file delivery (GET /media/file/{filename}). The nginx
// gateway routes /api/media* here and the /media/ location (rich-text <img>
// URLs) flips to this service too.
//
// Store split (Wave 2): "process split, shared store". The media service is its
// own process (MEDIA_PORT, default 8084) but connects to the same Postgres
// `omoikane` store as the other services. Since Phase 31 the trash/dashboard
// aggregators read via internal APIs — no service reads another's tables
// directly. Uploaded files stay on the shared bind-mounted disk (../backend:/app),
// so ./uploads resolves to the same backend/uploads directory every container sees.
//
// Events: media is the single writer of media events. It wires the Phase 28
// outbox (media upload enqueues inside the business transaction) and runs a
// Relay that publishes media.uploaded CloudEvents to Kafka. Media is the only
// service with a non-nil outbox for these rows, so there is never double
// emission. Kafka being unreachable must not fatal the service: topic
// provisioning is best-effort and the relay keeps retrying pending rows.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"
	"omoikane-backend/internal/observability"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// JSON structured logs + process-wide Prometheus registry (Phase 34).
	observability.Setup("media")

	dsn := os.Getenv("MEDIA_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"
	}
	port := os.Getenv("MEDIA_PORT")
	if port == "" {
		port = "8084"
	}

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("media-service: failed to connect to database: %v", err)
	}

	// Migrate only the media tables owned (Wave 2: shared store) by media. The
	// outbox table ships with it.
	if err := db.AutoMigrate(&models.MediaItem{}); err != nil {
		log.Fatalf("media-service: failed to migrate media tables: %v", err)
	}
	outbox := events.NewGormOutboxStore(db)
	if err := events.MigrateOutbox(db); err != nil {
		log.Fatalf("media-service: failed to migrate outbox: %v", err)
	}
	log.Println("media-service connected and migrated (shared omoikane store + outbox)")

	// Migration-only mode (Phase 34): see cmd/auth for the rationale.
	if os.Getenv("MIGRATE_ONLY") == "1" {
		log.Println("media-service: migrations complete, exiting (MIGRATE_ONLY=1)")
		return
	}

	// Events wiring. EnsureTopics is best-effort: if Kafka is down the relay
	// simply keeps retrying pending outbox rows; the service stays up.
	eventsCfg := events.ConfigFromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: media-service: ensure kafka topics: %v (relay will retry)", err)
	}
	producer, err := events.NewProducer(eventsCfg)
	if err != nil {
		log.Printf("WARNING: media-service: kafka producer disabled (%v); outbox rows stay pending", err)
		producer = nil
	}
	if producer != nil {
		defer producer.Close()
	}

	h := &handlers.Handler{
		DB:              db,
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		UploadDir:       getEnv("UPLOAD_DIR", "./uploads"),
		MediaBaseURL:    getEnv("MEDIA_BASE_URL", ""),
		// Media owns the "media" trash entity (Phase 31); hard-delete keeps the
		// disk cleanup in this service.
		TrashEntities: []string{"media"},
		// Media is the single writer of media events (media.uploaded). The
		// outbox store is transactional (handler enqueues inside the business
		// DB tx); the relay below flushes it to Kafka.
		Outbox: outbox,
	}

	// Outbox relay: publishes media.uploaded to Kafka on an interval until the
	// process shuts down.
	if producer != nil {
		relay := events.NewRelay(outbox, producer, eventsCfg)
		go relay.Run(ctx)
	} else {
		log.Println("WARNING: media-service: outbox relay not started (no producer)")
	}

	mux := newMediaMux(h, getEnv("INTERNAL_TOKEN", ""))

	addr := ":" + port
	log.Printf("media-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("media-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("media-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("media-service: shutdown error: %v", err)
	}
}

// newMediaMux registers every media-service route. Mux paths carry NO /api
// prefix — nginx prefix locations strip the prefix, so /api/media -> /media.
// Exposed as a function so cmd/media tests can exercise the full wiring.
// internalToken authenticates the /internal/* endpoints (trash aggregator +
// dashboard facade) via the shared X-Internal-Token header.
func newMediaMux(h *handlers.Handler, internalToken string) *http.ServeMux {
	mux := http.NewServeMux()

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Prometheus scrape endpoint (Phase 34); service-level, never gateway-exposed.
	mux.Handle("GET /metrics", observability.MetricsHandler())

	// Public CDN-ready file delivery (rich-text <img src="/media/file/...">).
	mux.HandleFunc("GET /media/file/{filename}", h.ServeMediaFile)

	// Media CRUD (all auth-protected; no public cache on these).
	mux.HandleFunc("GET /media", h.Auth(h.GetMedia))
	mux.HandleFunc("POST /media", h.Auth(h.UploadMedia))
	mux.HandleFunc("GET /media/{id}", h.Auth(h.GetMediaItem))
	mux.HandleFunc("PUT /media/{id}", h.Auth(h.UpdateMedia))
	mux.HandleFunc("DELETE /media/{id}", h.Auth(h.DeleteMedia))
	mux.HandleFunc("POST /media/batch", h.Auth(h.BatchMedia))

	// Internal endpoints (Phase 31): served to the trash aggregator + dashboard
	// facade inside the compose network. Guarded by the shared internal token;
	// the gateway never exposes /internal/*.
	mux.HandleFunc("GET /internal/stats", middleware.InternalAuth(internalToken, h.InternalStatsMedia))
	mux.HandleFunc("GET /internal/trash", middleware.InternalAuth(internalToken, h.InternalTrashList))
	mux.HandleFunc("GET /internal/trash/count", middleware.InternalAuth(internalToken, h.InternalTrashCount))
	mux.HandleFunc("POST /internal/trash/{entity}/{id}/restore", middleware.InternalAuth(internalToken, h.InternalTrashRestore))
	mux.HandleFunc("DELETE /internal/trash/{entity}/{id}", middleware.InternalAuth(internalToken, h.InternalTrashHardDelete))
	mux.HandleFunc("DELETE /internal/trash", middleware.InternalAuth(internalToken, h.InternalTrashEmpty))

	return mux
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
