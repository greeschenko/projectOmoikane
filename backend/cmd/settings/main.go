// Command settings is the Omoikane settings service (Phase 31, Wave 3).
//
// It owns the site-settings surface that was previously served by the monolith
// (cmd/api): the public GET /settings read (site name/tagline/logo/favicon/email
// templates) and the admin PUT /settings write. The nginx gateway routes
// /api/settings here — but NOT /api/settings/profile|password, which are auth
// identity routes held by auth-service (longest-prefix matching keeps them
// separate in nginx).
//
// Store split (Wave 3): "process split, shared store". The settings service is
// its own process (SETTINGS_PORT, default 8086) but connects to the same
// Postgres `omoikane` store as the other services (SiteSetting row id=1), so a
// physical schema partition is deferred.
//
// The public GET /settings is cached (CacheRead, 30s) and the PUT flushes the
// shared Redis instance, so settings writes invalidate the same cache the
// content service and any SSR reads share.
//
// Events: settings has no outbound events in Phase 31. The outbox table + relay
// are wired now so the service is event-ready; Kafka being unreachable must not
// fatal the service.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omoikane-backend/internal/cache"
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
	observability.Setup("settings")

	dsn := os.Getenv("SETTINGS_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"
	}
	port := os.Getenv("SETTINGS_PORT")
	if port == "" {
		port = "8086"
	}

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("settings-service: failed to connect to database: %v", err)
	}

	// Migrate the site settings table (auth-service also migrates it for the
	// forgot-password template read; AutoMigrate is idempotent). The outbox
	// table ships with it.
	if err := db.AutoMigrate(&models.SiteSetting{}); err != nil {
		log.Fatalf("settings-service: failed to migrate settings tables: %v", err)
	}
	outbox := events.NewGormOutboxStore(db)
	if err := events.MigrateOutbox(db); err != nil {
		log.Fatalf("settings-service: failed to migrate outbox: %v", err)
	}
	log.Println("settings-service connected and migrated (shared omoikane store + outbox)")

	// Migration-only mode (Phase 34): see cmd/auth for the rationale.
	if os.Getenv("MIGRATE_ONLY") == "1" {
		log.Println("settings-service: migrations complete, exiting (MIGRATE_ONLY=1)")
		return
	}

	// Events wiring. EnsureTopics is best-effort: if Kafka is down the relay
	// simply keeps retrying pending outbox rows; the service stays up.
	eventsCfg := events.ConfigFromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg.Brokers, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: settings-service: ensure kafka topics: %v (relay will retry)", err)
	}
	producer, err := events.NewProducer(eventsCfg)
	if err != nil {
		log.Printf("WARNING: settings-service: kafka producer disabled (%v); outbox rows stay pending", err)
		producer = nil
	}
	if producer != nil {
		defer producer.Close()
	}

	h := &handlers.Handler{
		DB:              db,
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		// Settings.updated events flow through the outbox relay to Kafka.
		Outbox: outbox,
	}

	// The public GET /settings is cached and settings writes must invalidate
	// the shared Redis instance (SSR reads + the content service read the same
	// settings-derived values).
	var c cache.Cache = cache.NoopCache{}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		if rc, rerr := cache.NewRedis(redisURL, 30*time.Second); rerr != nil {
			log.Printf("WARNING: settings-service: cache disabled (%v)", rerr)
		} else {
			c = rc
		}
	}
	h.Cache = c

	// Outbox relay: keeps the (currently empty) outbox drained until the
	// process shuts down.
	if producer != nil {
		relay := events.NewRelay(outbox, producer, eventsCfg)
		go relay.Run(ctx)
	} else {
		log.Println("WARNING: settings-service: outbox relay not started (no producer)")
	}

	mux := newSettingsMux(h)

	addr := ":" + port
	log.Printf("settings-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("settings-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("settings-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("settings-service: shutdown error: %v", err)
	}
}

// newSettingsMux registers every settings-service route. Mux paths carry NO /api
// prefix — nginx prefix locations strip the prefix, so /api/settings -> /settings.
// Exposed as a function so cmd/settings tests can exercise the full wiring.
func newSettingsMux(h *handlers.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	cacheTTL := 30 * time.Second

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Prometheus scrape endpoint (Phase 34); service-level, never gateway-exposed.
	mux.Handle("GET /metrics", observability.MetricsHandler())

	// Site settings: public cached read + admin write (flushes the shared cache)
	mux.HandleFunc("GET /settings", middleware.CacheRead(h.Cache, cacheTTL, h.GetSettings))
	mux.HandleFunc("PUT /settings", h.Admin(h.UpdateSettings))

	return mux
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
