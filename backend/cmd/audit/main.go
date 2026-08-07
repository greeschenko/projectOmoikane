package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"omoikane-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", handleReceiveEvent)
	mux.HandleFunc("GET /logs", handleGetLogs)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := ":" + port
	log.Printf("Audit service starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Audit service failed: %v", err)
	}
}

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
