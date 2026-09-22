// Command auth is the Omoikane auth service (Phase 29, Wave 1).
//
// It owns the identity surface that was previously served by the monolith
// (cmd/api): setup, login/register/forgot-password, profiles, users, and API
// tokens. The nginx gateway routes /api/auth*, /api/users*, /api/api-tokens*,
// /api/setup* and the /api/settings/profile|password paths here.
//
// Store split (Wave 1): "process split, shared store". The auth service is its
// own process (AUTH_PORT, default 8082) but connects to the same Postgres
// `omoikane` store as the other services. Since Phase 31 the trash/dashboard
// aggregators read via internal APIs — no service reads another's tables
// directly. A physical schema partition is deferred.
//
// Events: auth is the single writer of auth events. It wires the Phase 28
// outbox (User/ApiToken writes enqueue inside the business transaction) and
// runs a Relay that publishes user.registered CloudEvents to Kafka. Auth is
// the only service with a non-nil outbox for these rows so there is never
// double emission. Kafka being unreachable must not fatal the service: topic
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

	"omoikane-backend/internal/cache"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := os.Getenv("AUTH_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"
	}
	port := os.Getenv("AUTH_PORT")
	if port == "" {
		port = "8082"
	}

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("auth-service: failed to connect to database: %v", err)
	}

	// Migrate only the identity tables owned (Wave 1: shared store) by auth:
	// users, API tokens, password reset tokens and SiteSetting (forgot-password
	// reads the email template settings). The outbox table ships with it.
	if err := db.AutoMigrate(
		&models.User{},
		&models.ApiToken{},
		&models.PasswordResetToken{},
		&models.SiteSetting{},
	); err != nil {
		log.Fatalf("auth-service: failed to migrate auth tables: %v", err)
	}
	outbox := events.NewGormOutboxStore(db)
	if err := events.MigrateOutbox(db); err != nil {
		log.Fatalf("auth-service: failed to migrate outbox: %v", err)
	}
	log.Println("auth-service connected and migrated (shared omoikane store + outbox)")

	// Events wiring. EnsureTopics is best-effort: if Kafka is down the relay
	// simply keeps retrying pending outbox rows; the service stays up.
	eventsCfg := events.ConfigFromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg.Brokers, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: auth-service: ensure kafka topics: %v (relay will retry)", err)
	}
	producer, err := events.NewProducer(eventsCfg)
	if err != nil {
		log.Printf("WARNING: auth-service: kafka producer disabled (%v); outbox rows stay pending", err)
		producer = nil
	}
	if producer != nil {
		defer producer.Close()
	}

	h := &handlers.Handler{
		DB:              db,
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		SMTPHost:        getEnv("SMTP_HOST", ""),
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUser:        getEnv("SMTP_USER", ""),
		SMTPPass:        getEnv("SMTP_PASS", ""),
		SMTPFrom:        getEnv("SMTP_FROM", "noreply@omoikane.local"),
		RecaptchaSecret: getEnv("RECAPTCHA_SECRET", ""),
		AuditServiceURL: getEnv("AUDIT_SERVICE_URL", ""),
		// Auth owns the "user" trash entity (Phase 31): its internal endpoints
		// serve only user rows to the trash aggregator.
		TrashEntities: []string{"user"},
		// Auth is the single writer of auth events. The outbox store is
		// transactional (handler enqueues inside the business DB tx); the relay
		// below flushes it to Kafka.
		Outbox: outbox,
	}

	// User writes invalidate the shared public cache (blog lists render author
	// names), so auth joins the platform Redis instance and flushes on user
	// mutations — invalidating every service's SSR/cache tier.
	var c cache.Cache = cache.NoopCache{}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		if rc, rerr := cache.NewRedis(redisURL, 30*time.Second); rerr != nil {
			log.Printf("WARNING: auth-service: cache disabled (%v)", rerr)
		} else {
			c = rc
		}
	}
	h.Cache = c

	// Outbox relay: publishes user.registered to Kafka on an interval until the
	// process shuts down. When Kafka is down, publish fails -> rows are kept
	// pending (retried up to OutboxMaxAttempts before being flagged failed).
	if producer != nil {
		relay := events.NewRelay(outbox, producer, eventsCfg)
		go relay.Run(ctx)
	} else {
		log.Println("WARNING: auth-service: outbox relay not started (no producer)")
	}

	mux := newAuthMux(h, getEnv("INTERNAL_TOKEN", ""))

	addr := ":" + port
	log.Printf("auth-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("auth-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("auth-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("auth-service: shutdown error: %v", err)
	}
}

// newAuthMux registers every auth-service route. Mux paths carry NO /api prefix
// — nginx prefix locations strip the prefix, so /api/auth/login -> /auth/login.
// Exposed as a function so cmd/auth tests can exercise the full wiring.
// internalToken authenticates the /internal/* endpoints (trash aggregator +
// dashboard facade) via the shared X-Internal-Token header.
func newAuthMux(h *handlers.Handler, internalToken string) *http.ServeMux {
	mux := http.NewServeMux()

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Setup check (public; used by the frontend home + setup pages to decide
	// whether to render the installer)
	mux.HandleFunc("GET /setup/check", h.SetupStatus)
	mux.HandleFunc("POST /setup", h.Setup)

	// Auth (public)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/logout", h.Logout)
	forgotPasswordLimiter := middleware.NewRateLimiter(1.0/300.0, 3, 15*time.Minute)
	mux.HandleFunc("POST /auth/forgot-password", forgotPasswordLimiter.Middleware(h.ForgotPassword))
	mux.HandleFunc("POST /auth/reset-password", h.ResetPassword)

	// Settings — only the authenticated profile surface lives here; the public
	// GET/PUT /settings route belongs to the settings-service (Phase 31).
	mux.HandleFunc("GET /settings/profile", h.Auth(h.GetProfile))
	mux.HandleFunc("PUT /settings/profile", h.Auth(h.UpdateProfile))
	mux.HandleFunc("POST /settings/password", h.Auth(h.ChangePassword))

	// Users (admin only)
	mux.HandleFunc("GET /users", h.Admin(h.GetUsers))
	mux.HandleFunc("POST /users", h.Admin(h.CreateUser))
	mux.HandleFunc("PUT /users/{id}", h.Admin(h.UpdateUser))
	mux.HandleFunc("DELETE /users/{id}", h.Admin(h.DeleteUser))
	mux.HandleFunc("POST /users/batch", h.Admin(h.BatchUsers))

	// API tokens (admin only, for headless CMS access)
	mux.HandleFunc("GET /api-tokens", h.Admin(h.GetApiTokens))
	mux.HandleFunc("POST /api-tokens", h.Admin(h.CreateApiToken))
	mux.HandleFunc("DELETE /api-tokens/{id}", h.Admin(h.DeleteApiToken))

	// Internal endpoints (Phase 31): served to the trash aggregator and the
	// dashboard facade inside the compose network. Guarded by the shared
	// internal token; the gateway never exposes /internal/*.
	mux.HandleFunc("GET /internal/stats", middleware.InternalAuth(internalToken, h.InternalStatsUsers))
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
