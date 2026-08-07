package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"
)

type createMessageRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// GetMessages returns messages plus the current user's unread count.
// @Summary List messages
// @Description Returns all messages plus the number of messages unread by the current user.
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /messages [get]
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var messages []models.Message
	h.DB.Order("created_at desc").Find(&messages)

	userID := middleware.GetUserID(r)

	result := make([]map[string]interface{}, 0)
	unreadCount := 0
	for _, m := range messages {
		item := sanitizeMessageJSON(m)
		readBy := parseReadBy(m.ReadBy)
		item["readBy"] = readBy
		read := false
		for _, uid := range readBy {
			if uid == userID {
				read = true
				break
			}
		}
		if !read {
			unreadCount++
		}
		result = append(result, item)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages":     result,
		"unreadCount":  unreadCount,
	})
}

// GetMessage returns a single message by ID.
// @Summary Get message
// @Description Returns a single message by its numeric ID.
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Param id path int true "Message ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /messages/{id} [get]
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

	item := sanitizeMessageJSON(msg)
	item["readBy"] = parseReadBy(msg.ReadBy)

	json.NewEncoder(w).Encode(item)
}

// CreateMessage creates a new message (admin only).
// @Summary Create message
// @Description Creates a new message visible to authenticated users.
// @Tags messages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createMessageRequest true "Message details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /messages [post]
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
	item["readBy"] = []uint{}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

// MarkRead marks a message as read by the current user.
// @Summary Mark message read
// @Description Marks a message as read for the current user.
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Param id path int true "Message ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /messages/{id}/read [post]
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
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
	}

	readBy = append(readBy, userID)
	data, _ := json.Marshal(readBy)
	h.DB.Model(&msg).Update("read_by", string(data))

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteMessage soft-deletes a message (admin only).
// @Summary Delete message
// @Description Soft-deletes a message; it moves to trash and can be restored.
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Param id path int true "Message ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /messages/{id} [delete]
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

// MarkAllRead marks all messages as read for the current user.
// @Summary Mark all messages read
// @Description Marks every message as read for the current user.
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]bool
// @Router /messages/read-all [post]
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := middleware.GetUserID(r)

	var messages []models.Message
	h.DB.Find(&messages)

	for _, msg := range messages {
		readBy := parseReadBy(msg.ReadBy)
		found := false
		for _, uid := range readBy {
			if uid == userID {
				found = true
				break
			}
		}
		if !found {
			readBy = append(readBy, userID)
			data, _ := json.Marshal(readBy)
			h.DB.Model(&msg).Update("read_by", string(data))
		}
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteAllMessages permanently deletes all messages (admin only).
// @Summary Delete all messages
// @Description Hard-deletes every message row.
// @Tags messages
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]bool
// @Router /messages [delete]
func (h *Handler) DeleteAllMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	h.DB.Exec("DELETE FROM messages")
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

func parseReadBy(s string) []uint {
	if s == "" || s == "[]" {
		return []uint{}
	}
	var ids []uint
	if err := json.Unmarshal([]byte(s), &ids); err != nil {
		return []uint{}
	}
	return ids
}
