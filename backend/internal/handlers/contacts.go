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

// SubmitContact submits the public contact form.
// @Summary Submit contact form
// @Description Stores a contact form submission. Requires a valid reCAPTCHA token when reCAPTCHA is configured.
// @Tags contacts
// @Accept json
// @Produce json
// @Param body body contactRequest true "Contact form data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /contact [post]
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

// GetContacts returns contact form submissions plus unread count (admin only).
// @Summary List contact submissions
// @Description Returns all contact form messages, newest first, plus the number of unread ones.
// @Tags contacts
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /contacts [get]
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

// GetContact returns a single contact submission (admin only).
// @Summary Get contact submission
// @Description Returns a single contact form message by ID.
// @Tags contacts
// @Produce json
// @Security BearerAuth
// @Param id path int true "Contact ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /contacts/{id} [get]
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

// MarkContactRead marks a contact submission as read (admin only).
// @Summary Mark contact read
// @Description Marks a contact form message as read.
// @Tags contacts
// @Produce json
// @Security BearerAuth
// @Param id path int true "Contact ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /contacts/{id}/read [post]
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

// DeleteContact soft-deletes a contact submission (admin only).
// @Summary Delete contact submission
// @Description Soft-deletes a contact form message; it moves to trash and can be restored.
// @Tags contacts
// @Produce json
// @Security BearerAuth
// @Param id path int true "Contact ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /contacts/{id} [delete]
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
