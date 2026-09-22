// Command messages is the Omoikane messages service (Phase 31, Wave 3).
//
// It owns the messaging surface that was previously served by the monolith
// (cmd/api): broadcast messages (admin) and the public contact form + admin
// contact inbox. The nginx gateway routes /api/contact*, /api/contacts* and
// /api/messages* here.
//
// Store split (Wave 3): "process split, shared store". The messages service is
// its own process (MESSAGES_PORT, default 8085) but connects to the same
// Postgres `omoikane` store as the other services. Since Phase 31 the
// trash/dashboard aggregators read via internal APIs — no service reads
// another's tables directly. A physical schema partition is deferred.
//
// No public GET surface exists (messages/contacts are all auth-protected), so
// this service does NOT wire the shared Redis cache.
//
// Events: messages has no outbound events in Phase 31 (contact.received is a
// Phase 32 emission). The outbox table + relay are wired now so the service is
// event-ready; Kafka being unreachable must not fatal the service.
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

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := os.Getenv("MESSAGES_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane sslmode=disable"
	}
	port := os.Getenv("MESSAGES_PORT")
	if port == "" {
		port = "8085"
	}

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("messages-service: failed to connect to database: %v", err)
	}

	// Migrate only the messaging tables owned (Wave 3: shared store) by
	// messages: the broadcast message board and the contact inbox. The outbox
	// table ships with it.
	if err := db.AutoMigrate(
		&models.Message{},
		&models.ContactMessage{},
	); err != nil {
		log.Fatalf("messages-service: failed to migrate messages tables: %v", err)
	}
	outbox := events.NewGormOutboxStore(db)
	if err := events.MigrateOutbox(db); err != nil {
		log.Fatalf("messages-service: failed to migrate outbox: %v", err)
	}
	log.Println("messages-service connected and migrated (shared omoikane store + outbox)")

	// Events wiring. EnsureTopics is best-effort: if Kafka is down the relay
	// simply keeps retrying pending outbox rows; the service stays up.
	eventsCfg := events.ConfigFromEnv()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg.Brokers, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: messages-service: ensure kafka topics: %v (relay will retry)", err)
	}
	producer, err := events.NewProducer(eventsCfg)
	if err != nil {
		log.Printf("WARNING: messages-service: kafka producer disabled (%v); outbox rows stay pending", err)
		producer = nil
	}
	if producer != nil {
		defer producer.Close()
	}

	h := &handlers.Handler{
		DB:              db,
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		RecaptchaSecret: getEnv("RECAPTCHA_SECRET", ""),
		AuditServiceURL: getEnv("AUDIT_SERVICE_URL", ""),
		// Messages owns the contact/message trash entities (Phase 31): its
		// internal endpoints serve only those rows to the trash aggregator.
		TrashEntities: []string{"contact", "message"},
		// No outbound events yet (Phase 32 adds contact.received); the outbox
		// stays wired so the service is event-ready.
		Outbox: outbox,
	}

	// Outbox relay: keeps the (currently empty) outbox drained until the
	// process shuts down.
	if producer != nil {
		relay := events.NewRelay(outbox, producer, eventsCfg)
		go relay.Run(ctx)
	} else {
		log.Println("WARNING: messages-service: outbox relay not started (no producer)")
	}

	mux := newMessagesMux(h, getEnv("INTERNAL_TOKEN", ""))

	addr := ":" + port
	log.Printf("messages-service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("messages-service failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("messages-service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("messages-service: shutdown error: %v", err)
	}
}

// newMessagesMux registers every messages-service route. Mux paths carry NO /api
// prefix — nginx prefix locations strip the prefix, so /api/contacts -> /contacts.
// Exposed as a function so cmd/messages tests can exercise the full wiring.
// internalToken authenticates the /internal/* endpoints (trash aggregator +
// dashboard facade) via the shared X-Internal-Token header.
func newMessagesMux(h *handlers.Handler, internalToken string) *http.ServeMux {
	mux := http.NewServeMux()

	// Health (readiness check from Makefile / compose healthcheck)
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Contact form (public; ReCAPTCHA enforced by the handler)
	mux.HandleFunc("POST /contact", h.SubmitContact)

	// Contacts (admin only)
	mux.HandleFunc("GET /contacts", h.Admin(h.GetContacts))
	mux.HandleFunc("GET /contacts/{id}", h.Admin(h.GetContact))
	mux.HandleFunc("POST /contacts/{id}/read", h.Admin(h.MarkContactRead))
	mux.HandleFunc("DELETE /contacts/{id}", h.Admin(h.DeleteContact))

	// Broadcast messages (authed reads, admin writes)
	mux.HandleFunc("GET /messages", h.Auth(h.GetMessages))
	mux.HandleFunc("POST /messages", h.Admin(h.CreateMessage))
	mux.HandleFunc("GET /messages/{id}", h.Auth(h.GetMessage))
	mux.HandleFunc("POST /messages/{id}/read", h.Auth(h.MarkRead))
	mux.HandleFunc("POST /messages/read-all", h.Auth(h.MarkAllRead))
	mux.HandleFunc("DELETE /messages", h.Admin(h.DeleteAllMessages))
	mux.HandleFunc("DELETE /messages/{id}", h.Admin(h.DeleteMessage))

	// Internal endpoints (Phase 31): served to the trash aggregator and the
	// dashboard facade inside the compose network. Guarded by the shared
	// internal token; the gateway never exposes /internal/*.
	mux.HandleFunc("GET /internal/stats", middleware.InternalAuth(internalToken, h.InternalStatsMessages))
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
