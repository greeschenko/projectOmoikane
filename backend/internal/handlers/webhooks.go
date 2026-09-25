package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"
)

// webhookEventTypes is the fixed set of subscription event types (Phase 35).
// It mirrors the 7 live backbone types (see backend/docs/events.md); a future
// workflow module can extend this list. Creating a subscription against any
// other type is rejected with 400 so a typo never silently subscribes to the
// wrong stream.
var webhookEventTypes = []string{
	events.TypeUserRegistered,
	events.TypeAuthLogin,
	events.TypePagePublished,
	events.TypePostPublished,
	events.TypeMediaUploaded,
	events.TypeContactReceived,
	events.TypeSettingsUpdated,
}

// TypeWebhookPing is the synthetic event type for POST /webhooks/{id}/test
// (not a backbone event — it only exists in the delivery log).
const TypeWebhookPing = "org.omoikane.webhooks.ping.v1"

func validWebhookEventType(t string) bool {
	for _, ty := range webhookEventTypes {
		if ty == t {
			return true
		}
	}
	return false
}

type createWebhookRequest struct {
	EventType string `json:"eventType"`
	URL       string `json:"url"`
	// Secret is optional: when omitted the service generates one (shown once).
	Secret string `json:"secret,omitempty"`
	Active *bool  `json:"active,omitempty"`
}

type updateWebhookRequest struct {
	EventType *string `json:"eventType,omitempty"`
	URL       *string `json:"url,omitempty"`
	// Secret, when set (even to ""), rotates the signing key.
	Secret *string `json:"secret,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

// webhookJSON renders a subscription WITHOUT the secret (never leaks it after
// creation). The create response additionally carries the one-time raw secret
// (see CreateWebhook).
func webhookJSON(s models.WebhookSubscription) map[string]interface{} {
	return map[string]interface{}{
		"id":        s.ID,
		"eventType": s.EventType,
		"url":       s.URL,
		"active":    s.Active,
		"createdAt": s.CreatedAt,
		"updatedAt": s.UpdatedAt,
	}
}

// generateWebhookSecret returns a 32-byte URL-safe random key (crypto/rand).
func generateWebhookSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate webhook secret: %w", err)
	}
	return encodeBase64URL(buf), nil
}

// GetWebhooks lists webhook subscriptions (admin only).
// @Summary List webhook subscriptions
// @Description Returns all webhook subscriptions (event type, URL, active flag, timestamps). Secrets are never included.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{}
// @Router /webhooks [get]
func (h *Handler) GetWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var subs []models.WebhookSubscription
	h.DB.Order("created_at desc").Find(&subs)

	result := make([]map[string]interface{}, 0, len(subs))
	for _, s := range subs {
		result = append(result, webhookJSON(s))
	}
	json.NewEncoder(w).Encode(result)
}

// CreateWebhook creates a webhook subscription (admin only). The raw HMAC
// secret is returned exactly once; subsequent reads never include it.
// @Summary Create webhook subscription
// @Description Creates a subscription that delivers one event type to a URL. If no secret is supplied a random HMAC key is generated and returned only once.
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createWebhookRequest true "Event type, destination URL and optional HMAC secret"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /webhooks [post]
func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validWebhookEventType(req.EventType) {
		writeJSONError(w, http.StatusBadRequest, "unknown eventType (allowed: the 7 live backbone types)")
		return
	}
	if req.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}

	secret := req.Secret
	if secret == "" {
		generated, err := generateWebhookSecret()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		secret = generated
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	sub := models.WebhookSubscription{
		EventType: req.EventType,
		URL:       req.URL,
		Secret:    secret,
		Active:    active,
	}
	if err := h.DB.Create(&sub).Error; err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create webhook subscription")
		return
	}

	result := webhookJSON(sub)
	result["secret"] = secret // one-time reveal
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// GetWebhook returns a single webhook subscription (admin only).
// @Summary Get webhook subscription
// @Description Returns one subscription by id (event type, URL, active flag, timestamps).
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Subscription ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /webhooks/{id} [get]
func (h *Handler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := parsePathID(r)
	var sub models.WebhookSubscription
	if err := h.DB.First(&sub, id).Error; err != nil {
		writeJSONError(w, http.StatusNotFound, "subscription not found")
		return
	}
	json.NewEncoder(w).Encode(webhookJSON(sub))
}

// UpdateWebhook updates a webhook subscription (admin only). A non-empty
// secret rotates the signing key; active toggles delivery.
// @Summary Update webhook subscription
// @Description Updates event type, URL, active flag and optionally rotates the HMAC secret.
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Subscription ID"
// @Param body body updateWebhookRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /webhooks/{id} [put]
func (h *Handler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := parsePathID(r)
	var sub models.WebhookSubscription
	if err := h.DB.First(&sub, id).Error; err != nil {
		writeJSONError(w, http.StatusNotFound, "subscription not found")
		return
	}

	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.EventType != nil {
		if !validWebhookEventType(*req.EventType) {
			writeJSONError(w, http.StatusBadRequest, "unknown eventType (allowed: the 7 live backbone types)")
			return
		}
		updates["event_type"] = *req.EventType
	}
	if req.URL != nil {
		if *req.URL == "" {
			writeJSONError(w, http.StatusBadRequest, "url cannot be empty")
			return
		}
		updates["url"] = *req.URL
	}
	if req.Secret != nil {
		updates["secret"] = *req.Secret
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if len(updates) == 0 {
		writeJSONError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	if err := h.DB.Model(&sub).Updates(updates).Error; err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update webhook subscription")
		return
	}

	// Reload for a consistent response (Updates with a map does not refresh
	// the struct's fields).
	h.DB.First(&sub, id)
	json.NewEncoder(w).Encode(webhookJSON(sub))
}

// DeleteWebhook deletes a webhook subscription (admin only). Past delivery
// rows are retained (they keep their subscriptionId for the delivery log).
// @Summary Delete webhook subscription
// @Description Deletes a webhook subscription; past delivery log rows are retained.
// @Tags webhooks
// @Security BearerAuth
// @Param id path int true "Subscription ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /webhooks/{id} [delete]
func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := parsePathID(r)
	var sub models.WebhookSubscription
	if err := h.DB.First(&sub, id).Error; err != nil {
		writeJSONError(w, http.StatusNotFound, "subscription not found")
		return
	}
	if err := h.DB.Delete(&sub).Error; err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete webhook subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TestWebhook enqueues a pending delivery for a synthetic ping event (admin
// only). The delivery pump picks it up on its next poll, so a test ping flows
// through the exact same signature/retry machinery as a real delivery.
// @Summary Send test ping
// @Description Enqueues a synthetic smoke-test delivery to the subscription URL through the normal delivery pipeline.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Subscription ID"
// @Success 201 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /webhooks/{id}/test [post]
func (h *Handler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := parsePathID(r)
	var sub models.WebhookSubscription
	if err := h.DB.First(&sub, id).Error; err != nil {
		writeJSONError(w, http.StatusNotFound, "subscription not found")
		return
	}
	if !sub.Active {
		writeJSONError(w, http.StatusBadRequest, "subscription is inactive")
		return
	}

	now := time.Now().UTC()
	payload, _ := json.Marshal(events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      "webhooks-service",
		Type:        TypeWebhookPing,
		Subject:     fmt.Sprintf("subscription/%d", sub.ID),
		Time:        now,
		Data:        json.RawMessage(`{"message":"test ping from Omoikane admin"}`),
	})
	delivery := models.WebhookDelivery{
		SubscriptionID: sub.ID,
		EventID:        payloadEventID(payload),
		EventType:      TypeWebhookPing,
		Payload:        string(payload),
		Status:         "pending",
		NextAttemptAt:  now,
	}
	if err := h.DB.Create(&delivery).Error; err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to enqueue test ping")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":             delivery.ID,
		"subscriptionId": delivery.SubscriptionID,
		"eventType":      delivery.EventType,
		"status":         delivery.Status,
	})
}

// GetWebhookDeliveries lists delivery log rows (admin only) with optional
// status / eventType / subscriptionId filters and pagination.
// @Summary List webhook deliveries
// @Description Returns the delivery log (attempts, HTTP status, next retry). Filters: status, eventType, subscriptionId.
// @Tags webhooks
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (pending|delivered|failed|expired)"
// @Param eventType query string false "Filter by event type"
// @Param subscriptionId query int false "Filter by subscription"
// @Param limit query int false "Max results (1-500, default 100)"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} map[string]interface{}
// @Router /webhooks/deliveries [get]
func (h *Handler) GetWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := h.DB.Model(&models.WebhookDelivery{}).Order("created_at DESC")
	if status := r.URL.Query().Get("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if eventType := r.URL.Query().Get("eventType"); eventType != "" {
		q = q.Where("event_type = ?", eventType)
	}
	if subIDStr := r.URL.Query().Get("subscriptionId"); subIDStr != "" {
		if sid, err := strconv.ParseUint(subIDStr, 10, 64); err == nil {
			q = q.Where("subscription_id = ?", sid)
		}
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
	q.Count(&total)

	var rows []models.WebhookDelivery
	q.Limit(limit).Offset(offset).Find(&rows)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"deliveries": rows,
		"total":      total,
	})
}

// parsePathID reads the {id} path value (Go 1.22+ mux). Returns 0 when the
// segment is missing or not numeric.
func parsePathID(r *http.Request) uint {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// encodeBase64URL encodes raw bytes URL-safe without padding (compact secret).
func encodeBase64URL(b []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	dst := make([]byte, len(b))
	for i, c := range b {
		// Every byte maps onto one base64url-ish character; no padding needed.
		dst[i] = alphabet[int(c)%len(alphabet)]
	}
	return string(dst)
}

// payloadEventID extracts the CloudEvent id from an already-marshalled
// payload, or falls back to the raw payload hash — lets the test ping row
// carry a stable unique event id for the (subscription, event) index.
func payloadEventID(payload []byte) string {
	var probe struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(payload, &probe) == nil && probe.ID != "" {
		return probe.ID
	}
	return fmt.Sprintf("ping-%d", time.Now().UnixNano())
}

// writeJSONError is a small helper for consistent JSON errors.
func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
