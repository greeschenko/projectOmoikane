package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "omoikane-backend/cmd/audit/docs"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"
	"omoikane-backend/internal/observability"

	"github.com/segmentio/kafka-go"
	httpSwagger "github.com/swaggo/http-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title Omoikane Audit Service
// @version 1.0
// @description Internal microservice that consumes CloudEvents from the Kafka backbone and stores them as audit log entries. Not meant for direct public use.
// @BasePath /api/audit
var db *gorm.DB

func main() {
	// JSON structured logs + process-wide Prometheus registry (Phase 34).
	observability.Setup("audit")

	dsn := os.Getenv("AUDIT_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_audit sslmode=disable"
	}
	port := os.Getenv("AUDIT_PORT")
	if port == "" {
		port = "8081"
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to audit database: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		log.Fatalf("Failed to migrate audit database: %v", err)
	}
	log.Println("Audit service connected and migrated")

	// Migration-only mode (Phase 34): the Helm chart runs each DB service as a
	// post-install migration Job with MIGRATE_ONLY=1 so schema deploys are
	// explicit and restart-safe. Exit cleanly once migrations are applied.
	if os.Getenv("MIGRATE_ONLY") == "1" {
		log.Println("audit-service: migrations complete, exiting (MIGRATE_ONLY=1)")
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	// Events wiring (Phase 32): audit write path is the Kafka backbone, NOT
	// the retired HTTP POST /events. EnsureTopics is best-effort — if Kafka is
	// down the consumer retries in a loop and the service stays up.
	eventsCfg := events.ConfigFromEnv()
	// Never replay the historical backbone into a fresh audit store: only
	// events published after this group's first start are ingested (ignored
	// once the group has committed offsets).
	eventsCfg.ConsumerStartOffset = kafka.LastOffset

	ctx, stop := signalContext()
	defer stop()

	if err := events.EnsureTopics(ctx, eventsCfg, []string{eventsCfg.Topic, eventsCfg.DLQTopic}); err != nil {
		log.Printf("WARNING: audit-service: ensure kafka topics: %v (consumer will retry)", err)
	}

	handler := &auditEventHandler{db: db}
	consumer, err := events.NewConsumer(eventsCfg, "audit", handler)
	if err != nil {
		log.Printf("WARNING: audit-service: kafka consumer disabled (%v); no events will be ingested", err)
		consumer = nil
	}

	mux := newAuditMux(jwtSecret)

	addr := ":" + port
	log.Printf("Audit service starting on %s", addr)
	srv := &http.Server{Addr: addr, Handler: observability.Middleware(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Audit service failed: %v", err)
		}
	}()

	if consumer != nil {
		go runConsumerLoop(ctx, consumer)
	}

	<-ctx.Done()
	log.Println("Audit service shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("audit-service: shutdown error: %v", err)
	}
	if consumer != nil {
		if err := consumer.Close(); err != nil {
			log.Printf("audit-service: consumer close error: %v", err)
		}
	}
}

// newAuditMux registers every audit-service route. Mux paths carry NO /api
// prefix — nginx prefix locations strip the prefix, so /api/audit-logs ->
// /audit-logs. Exposed as a function so cmd/audit tests can exercise the
// full wiring. The admin surface (gateway-facing) validates the admin JWT.
func newAuditMux(jwtSecret string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler())
	// Prometheus scrape endpoint (Phase 34); service-level, never gateway-exposed.
	mux.Handle("GET /metrics", observability.MetricsHandler())
	mux.HandleFunc("GET /logs", handleGetLogs)
	// Admin-only surface exposed directly to the gateway (Phase 31): the
	// frontend audit-log page calls /api/audit-logs with the admin session
	// cookie, and nginx routes it here (no monolith proxy).
	mux.HandleFunc("GET /audit-logs", middleware.AdminRequired(jwtSecret, handleAuditLogsAdmin))
	mux.HandleFunc("GET /health", handleHealth)
	return mux
}

// signalContext returns a context cancelled on SIGINT/SIGTERM.
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		cancel()
	}()
	return ctx, cancel
}

// runConsumerLoop runs the audit consumer until ctx is done, retrying with a
// short backoff on transient Kafka errors (broker restarts, rebalances) so the
// service never dies because the backbone hiccupped.
func runConsumerLoop(ctx context.Context, consumer *events.Consumer) {
	for {
		err := consumer.Run(ctx)
		if err == nil || ctx.Err() != nil {
			log.Println("audit-service: kafka consumer stopped")
			return
		}
		log.Printf("WARNING: audit-service: kafka consumer error: %v (retrying in 2s)", err)
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

// handleHealth reports audit service health.
// @Summary Health check
// @Description Returns the audit service status.
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAuditLogsAdmin is the gateway-facing admin wrapper around handleGetLogs
// (Phase 31). It carries the /audit-logs swagger annotations; the raw /logs
// endpoint remains for direct service calls.
// @Summary List audit logs (admin)
// @Description Returns stored audit log entries. Supports filters and pagination (limit <= 500).
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param entity query string false "Filter by entity type"
// @Param action query string false "Filter by action"
// @Param userId query int false "Filter by actor user ID"
// @Param search query string false "Search user name or detail"
// @Param limit query int false "Max results (1-500, default 100)"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} map[string]interface{}
// @Router /audit-logs [get]
func handleAuditLogsAdmin(w http.ResponseWriter, r *http.Request) {
	handleGetLogs(w, r)
}

// handleGetLogs returns stored audit logs with filters and pagination.
// @Summary List audit logs
// @Description Returns stored audit log entries. Supports filters and pagination (limit <= 500).
// @Tags audit
// @Produce json
// @Param entity query string false "Filter by entity type"
// @Param action query string false "Filter by action"
// @Param userId query int false "Filter by actor user ID"
// @Param search query string false "Search user name or detail"
// @Param limit query int false "Max results (1-500, default 100)"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} map[string]interface{}
// @Router /logs [get]
func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var logs []models.AuditLog
	q := db.Order("created_at DESC")

	if entityType := r.URL.Query().Get("entity"); entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}
	if action := r.URL.Query().Get("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if userIDStr := r.URL.Query().Get("userId"); userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			q = q.Where("user_id = ?", uid)
		}
	}
	if search := r.URL.Query().Get("search"); search != "" {
		q = q.Where("user_name ILIKE ? OR detail ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var total int64
	q.Model(&models.AuditLog{}).Count(&total)
	q.Limit(limit).Offset(offset).Find(&logs)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": total,
	})
}
