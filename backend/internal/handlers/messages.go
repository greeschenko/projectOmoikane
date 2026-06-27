package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"
)

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var messages []models.Message
	h.DB.Order("created_at desc").Find(&messages)

	userID := middleware.GetUserID(r)

	result := make([]map[string]interface{}, 0)
	for _, m := range messages {
		item := sanitizeMessageJSON(m)
		item["read"] = isMessageReadBy(m, userID)
		result = append(result, item)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": result,
	})
}

func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid message ID"})
		return
	}

	var msg models.Message
	if err := h.DB.First(&msg, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Message not found"})
		return
	}

	userID := middleware.GetUserID(r)
	item := sanitizeMessageJSON(msg)
	item["read"] = isMessageReadBy(msg, userID)

	json.NewEncoder(w).Encode(item)
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Title is required"})
		return
	}

	msg := models.Message{
		Title:   req.Title,
		Content: req.Content,
		ReadBy:  "[]",
	}
	h.DB.Create(&msg)

	item := sanitizeMessageJSON(msg)
	item["read"] = false

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid message ID"})
		return
	}

	var msg models.Message
	if err := h.DB.First(&msg, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Message not found"})
		return
	}

	userID := middleware.GetUserID(r)

	// Parse existing ReadBy
	var readBy []uint
	if msg.ReadBy != "" && msg.ReadBy != "[]" {
		json.Unmarshal([]byte(msg.ReadBy), &readBy)
	}

	// Check if already read
	for _, uid := range readBy {
		if uid == userID {
			json.NewEncoder(w).Encode(map[string]bool{"read": true})
			return
		}
	}

	readBy = append(readBy, userID)
	data, _ := json.Marshal(readBy)
	h.DB.Model(&msg).Update("read_by", string(data))

	json.NewEncoder(w).Encode(map[string]bool{"read": true})
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid message ID"})
		return
	}

	var msg models.Message
	if err := h.DB.First(&msg, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Message not found"})
		return
	}

	h.DB.Delete(&msg)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func sanitizeMessageJSON(m models.Message) map[string]interface{} {
	return map[string]interface{}{
		"id":        m.ID,
		"title":     m.Title,
		"content":   m.Content,
		"createdAt": m.CreatedAt,
		"updatedAt": m.UpdatedAt,
	}
}

func isMessageReadBy(m models.Message, userID uint) bool {
	if m.ReadBy == "" || m.ReadBy == "[]" {
		return false
	}
	var readBy []uint
	if err := json.Unmarshal([]byte(m.ReadBy), &readBy); err != nil {
		return false
	}
	for _, uid := range readBy {
		if uid == userID {
			return true
		}
	}
	return false
}
