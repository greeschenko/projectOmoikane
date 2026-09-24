// Command content is the Omoikane content service (Phase 30, Wave 2).
//
// It owns the pages + blog surface that was previously served by the monolith
// (cmd/api): pages (list/slug/detail/create/update/delete/reorder/batch), blog
// posts, tags and categories. The nginx gateway routes /api/pages* and
// /api/blog* here.
//
// Store split (Wave 2): "process split, shared store". The content service is
// its own process (CONTENT_PORT, default 8083) but connects to the same
// Postgres `omoikane` store as the other services. Since Phase 31 the
// trash/dashboard aggregators read via internal APIs — no service reads
// another's tables directly. A physical schema partition is deferred.
//
// Events: content is the single writer of content events. It wires the Phase 28
// outbox (page/post writes enqueue inside the business transaction) and runs a
// Relay that publishes page.published / post.published CloudEvents to Kafka.
// Content is the only service with a non-nil outbox for these rows, so there is
// never double emission. Kafka being unreachable must not fatal the service:
// topic provisioning is best-effort and the relay keeps retrying pending rows.
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
	observability.Setup("content")

	dsn := os.Getenv("CONTENT_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"
	}
	port := os.Getenv("CONTENT_PORT")
	if port == "" {
		port = "8083"
	}

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("content-service: failed to connect to database: %v", err)
	}

	// Migrate only the content tables owned (Wave 2: shared store) by content:
	// pages, blog posts, tags, categories, likes and the tag join model. The
	// outbox table ships with it.
	if err := db.AutoMigrate(
		&models.Page{},
		&models.BlogPost{},
		&models.Tag{},
		&models.BlogPostTag{},
		&models.Category{},
		&models.Like{},
	); err != nil {
		log.Fatalf("content-service: failed to migrate content tables: %v", err)
	}
	outbox := events.NewGormOutboxStore(db)
	if err := events.MigrateOutbox(db); err != nil {
		log.Fatalf("content-service: failed to migrate outbox: %v", err)
	}
	log.Println("content-service connected and migrated (shared omoikane store + outbox)")

	// Migration-only mode (Phase 34): see cmd/auth for the rationale.
	if os.Getenv("MIGRATE_ONLY") == "1" {
		log.Println("content-service: migrations complete, exiting (MIGRATE_ONLY=1)")
		return
	}

	// Events wiring. EnsureTopics is best-effort: if Kafka is down the relay
	// simply keeps retrying pending outbox rows; the service stays up.
	eventsCfg := events.ConfigFromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg.Brokers, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: content-service: ensure kafka topics: %v (relay will retry)", err)
	}
	producer, err := events.NewProducer(eventsCfg)
	if err != nil {
		log.Printf("WARNING: content-service: kafka producer disabled (%v); outbox rows stay pending", err)
		producer = nil
	}
	if producer != nil {
		defer producer.Close()
	}

	h := &handlers.Handler{
		DB:              db,
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		// Content owns the page/post/tag/category trash entities (Phase 31).
		TrashEntities: []string{"page", "post", "tag", "category"},
		// Content is the single writer of content events (page.published,
		// post.published). The outbox store is transactional (handler enqueues
		// inside the business DB tx); the relay below flushes it to Kafka.
		Outbox: outbox,
	}

	// Public GET caches share the platform Redis instance, so cache flushes
	// from any service keep SSR reads (through the gateway) and gateway reads
	// (content-service, settings-service, auth-service) mutually consistent.
	var c cache.Cache = cache.NoopCache{}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		if rc, rerr := cache.NewRedis(redisURL, 30*time.Second); rerr != nil {
			log.Printf("WARNING: content-service: cache disabled (%v)", rerr)
		} else {
			c = rc
		}
	}
	h.Cache = c

	// Outbox relay: publishes page.published/post.published to Kafka on an
	// interval until the process shuts down.
	if producer != nil {
		relay := events.NewRelay(outbox, producer, eventsCfg)
		go relay.Run(ctx)
	} else {
		log.Println("WARNING: content-service: outbox relay not started (no producer)")
	}

	mux := newContentMux(h, getEnv("INTERNAL_TOKEN", ""))

	addr := ":" + port
	log.Printf("content-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("content-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("content-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("content-service: shutdown error: %v", err)
	}
}

// newContentMux registers every content-service route. Mux paths carry NO /api
// prefix — nginx prefix locations strip the prefix, so /api/pages -> /pages.
// Exposed as a function so cmd/content tests can exercise the full wiring.
// internalToken authenticates the /internal/* endpoints (trash aggregator +
// dashboard facade) via the shared X-Internal-Token header.
func newContentMux(h *handlers.Handler, internalToken string) *http.ServeMux {
	mux := http.NewServeMux()
	cacheTTL := 30 * time.Second

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Prometheus scrape endpoint (Phase 34); service-level, never gateway-exposed.
	mux.Handle("GET /metrics", observability.MetricsHandler())

	// Pages
	mux.HandleFunc("GET /pages", middleware.CacheRead(h.Cache, cacheTTL, h.GetPages))
	mux.HandleFunc("GET /pages/slug/{slug}", middleware.CacheRead(h.Cache, cacheTTL, h.GetPageBySlug))
	mux.HandleFunc("GET /pages/{id}", h.GetPage)
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("PUT /pages/{id}", h.Auth(h.UpdatePage))
	mux.HandleFunc("DELETE /pages/{id}", h.Auth(h.DeletePage))
	mux.HandleFunc("POST /pages/batch", h.Auth(h.BatchPages))
	mux.HandleFunc("PUT /pages/reorder", h.Auth(h.ReorderPages))

	// Blog posts
	mux.HandleFunc("GET /blog/posts", middleware.CacheRead(h.Cache, cacheTTL, h.GetPosts))
	mux.HandleFunc("GET /admin/blog/posts", h.Admin(h.GetAdminPosts))
	mux.HandleFunc("GET /blog/posts/slug/{slug}", middleware.CacheRead(h.Cache, cacheTTL, h.GetPostBySlug))
	mux.HandleFunc("GET /blog/posts/{id}", h.GetPost)
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	mux.HandleFunc("PUT /blog/posts/{id}", h.Auth(h.UpdatePost))
	mux.HandleFunc("DELETE /blog/posts/{id}", h.Auth(h.DeletePost))
	mux.HandleFunc("POST /blog/posts/batch", h.Auth(h.BatchPosts))
	mux.HandleFunc("POST /blog/posts/{id}/like", h.Auth(h.ToggleLike))

	// Tags + categories
	mux.HandleFunc("GET /blog/tags", h.GetTags)
	mux.HandleFunc("POST /blog/tags", h.Admin(h.CreateTag))
	mux.HandleFunc("DELETE /blog/tags/{id}", h.Admin(h.DeleteTag))
	mux.HandleFunc("GET /blog/categories", h.GetCategories)
	mux.HandleFunc("POST /blog/categories", h.Admin(h.CreateCategory))
	mux.HandleFunc("DELETE /blog/categories/{id}", h.Admin(h.DeleteCategory))

	// Internal endpoints (Phase 31): served to the trash aggregator and the
	// dashboard facade inside the compose network. Guarded by the shared
	// internal token; the gateway never exposes /internal/*.
	mux.HandleFunc("GET /internal/stats", middleware.InternalAuth(internalToken, h.InternalStatsContent))
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
