package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "omoikane-backend/cmd/audit/docs"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	httpSwagger "github.com/swaggo/http-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title Omoikane Audit Service
// @version 1.0
// @description Internal microservice that receives and stores audit log events emitted by the main API. Not meant for direct public use.
// @BasePath /api/audit
var db *gorm.DB

func main() {
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

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler())
	mux.HandleFunc("POST /events", handleReceiveEvent)
	mux.HandleFunc("GET /logs", handleGetLogs)
	// Admin-only surface exposed directly to the gateway (Phase 31): the
	// frontend audit-log page calls /api/audit-logs with the admin session
	// cookie, and nginx routes it here (no monolith proxy).
	mux.HandleFunc("GET /audit-logs", middleware.AdminRequired(jwtSecret, handleAuditLogsAdmin))
	mux.HandleFunc("GET /health", handleHealth)

	addr := ":" + port
	log.Printf("Audit service starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Audit service failed: %v", err)
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

// handleReceiveEvent stores a single audit event.
// @Summary Ingest audit event
// @Description Stores an audit log event emitted by the main API.
// @Tags audit
// @Accept json
// @Produce json
// @Param body body models.AuditLog true "Audit event"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /events [post]
func handleReceiveEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var event models.AuditLog
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if event.UserName == "" {
		event.UserName = "system"
	}
	if event.Action == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "action is required"})
		return
	}
	if event.EntityType == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "entityType is required"})
		return
	}

	if err := db.Create(&event).Error; err != nil {
		log.Printf("[audit] failed to store event: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to store event"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
