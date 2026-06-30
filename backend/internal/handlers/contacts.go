package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"omoikane-backend/internal/models"
	"omoikane-backend/internal/recaptcha"
)

type contactRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Subject        string `json:"subject"`
	Message        string `json:"message"`
	RecaptchaToken string `json:"recaptchaToken,omitempty"`
}

func (h *Handler) SubmitContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Name == "" || req.Email == "" || req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name, email, and message are required"})
		return
	}

	if req.Subject == "" {
		req.Subject = "Contact Form Submission"
	}

	if h.RecaptchaSecret != "" {
		ok, err := recaptcha.VerifyToken(h.RecaptchaSecret, req.RecaptchaToken)
		if err != nil || !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "reCAPTCHA verification failed"})
			return
		}
	}

	msg := models.ContactMessage{
		Name:    req.Name,
		Email:   req.Email,
		Subject: req.Subject,
		Message: req.Message,
	}
	if err := h.DB.Create(&msg).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save message"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Your message has been received",
	})
}

func (h *Handler) GetContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var messages []models.ContactMessage
	h.DB.Order("created_at desc").Find(&messages)

	result := make([]map[string]interface{}, 0)
	for _, m := range messages {
		result = append(result, sanitizeContactJSON(m))
	}

	unreadCount := 0
	for _, m := range messages {
		if !m.Read {
			unreadCount++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"contacts":    result,
		"unreadCount": unreadCount,
	})
}

func (h *Handler) GetContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid contact ID"})
		return
	}

	var msg models.ContactMessage
	if err := h.DB.First(&msg, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Contact message not found"})
		return
	}

	json.NewEncoder(w).Encode(sanitizeContactJSON(msg))
}

func (h *Handler) MarkContactRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid contact ID"})
		return
	}

	if err := h.DB.Model(&models.ContactMessage{}).Where("id = ?", id).Update("read", true).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Contact message not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid contact ID"})
		return
	}

	var msg models.ContactMessage
	if err := h.DB.First(&msg, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Contact message not found"})
		return
	}

	h.DB.Delete(&msg)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func sanitizeContactJSON(m models.ContactMessage) map[string]interface{} {
	return map[string]interface{}{
		"id":        m.ID,
		"name":      m.Name,
		"email":     m.Email,
		"subject":   m.Subject,
		"message":   m.Message,
		"read":      m.Read,
		"createdAt": m.CreatedAt,
		"updatedAt": m.UpdatedAt,
	}
}
