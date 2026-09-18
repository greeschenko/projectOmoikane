package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"omoikane-backend/internal/models"
)

// GetAuditLogs returns audit log entries (admin only).
// The audit microservice owns the audit_logs table (events are emitted there),
// so this handler proxies the request to it and returns its JSON.
// @Summary List audit logs
// @Description Returns audit log entries from the audit microservice. Supports filters and pagination (limit <= 500).
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param entity query string false "Filter by entity type (e.g. user, page, post)"
// @Param action query string false "Filter by action (e.g. create, update, delete)"
// @Param userId query int false "Filter by actor user ID"
// @Param search query string false "Search user name or detail"
// @Param limit query int false "Max results (1-500, default 100)"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} map[string]interface{}
// @Router /audit-logs [get]
func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.AuditServiceURL != "" {
		h.proxyAuditLogs(w, r)
		return
	}

	// Fallback: read from the local DB (used when no audit service is configured).
	var logs []models.AuditLog
	q := h.DB.Order("created_at DESC")

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

// proxyAuditLogs forwards the current request to the audit microservice's
// GET /logs endpoint, preserving all query filters and returning its response.
func (h *Handler) proxyAuditLogs(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, h.AuditServiceURL+"/logs", nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to build audit request"})
		return
	}
	req.URL.RawQuery = r.URL.RawQuery

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "Audit service unavailable"})
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
