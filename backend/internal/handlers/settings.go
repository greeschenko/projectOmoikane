package handlers

import (
	"encoding/json"
	"net/http"

	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	settings := models.SiteSetting{}
	result := h.DB.First(&settings, 1)
	if result.Error != nil {
		settings = models.SiteSetting{
			SiteName:    "Omoikane",
			Tagline:     "A headless-ish CMS",
			BlogEnabled: true,
		}
		h.DB.Create(&settings)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"siteName":          settings.SiteName,
		"tagline":           settings.Tagline,
		"logo":              settings.Logo,
		"favicon":           settings.Favicon,
		"blogEnabled":       settings.BlogEnabled,
		"resetEmailSubject": settings.ResetEmailSubject,
		"resetEmailBodyHTML": settings.ResetEmailBodyHTML,
	})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		SiteName           *string `json:"siteName,omitempty"`
		Tagline            *string `json:"tagline,omitempty"`
		Logo               *string `json:"logo,omitempty"`
		Favicon            *string `json:"favicon,omitempty"`
		BlogEnabled        *bool   `json:"blogEnabled,omitempty"`
		ResetEmailSubject  *string `json:"resetEmailSubject,omitempty"`
		ResetEmailBodyHTML *string `json:"resetEmailBodyHTML,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	settings := models.SiteSetting{}
	result := h.DB.First(&settings, 1)
	if result.Error != nil {
		settings = models.SiteSetting{
			SiteName:    "Omoikane",
			Tagline:     "A headless-ish CMS",
			BlogEnabled: true,
		}
		h.DB.Create(&settings)
	}

	updates := map[string]interface{}{}
	if req.SiteName != nil {
		updates["site_name"] = *req.SiteName
	}
	if req.Tagline != nil {
		updates["tagline"] = *req.Tagline
	}
	if req.Logo != nil {
		updates["logo"] = *req.Logo
	}
	if req.Favicon != nil {
		updates["favicon"] = *req.Favicon
	}
	if req.BlogEnabled != nil {
		updates["blog_enabled"] = *req.BlogEnabled
	}
	if req.ResetEmailSubject != nil {
		updates["reset_email_subject"] = *req.ResetEmailSubject
	}
	if req.ResetEmailBodyHTML != nil {
		updates["reset_email_body_html"] = *req.ResetEmailBodyHTML
	}

	if len(updates) > 0 {
		h.DB.Model(&settings).Updates(updates)
	}

	h.DB.First(&settings, 1)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"siteName":          settings.SiteName,
		"tagline":           settings.Tagline,
		"logo":              settings.Logo,
		"favicon":           settings.Favicon,
		"blogEnabled":       settings.BlogEnabled,
		"resetEmailSubject": settings.ResetEmailSubject,
		"resetEmailBodyHTML": settings.ResetEmailBodyHTML,
	})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name   *string `json:"name,omitempty"`
		Email  *string `json:"email,omitempty"`
		Avatar *string `json:"avatar,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	userID := middleware.GetUserID(r)
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		if *req.Name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Name cannot be empty"})
			return
		}
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	if len(updates) > 0 {
		h.DB.Model(&user).Updates(updates)
	}

	h.DB.First(&user, userID)
	json.NewEncoder(w).Encode(sanitizeUserJSON(user))
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Current and new password required"})
		return
	}

	if errMsg := validatePassword(req.NewPassword); errMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		return
	}

	userID := middleware.GetUserID(r)
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Current password is incorrect"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to hash password"})
		return
	}

	h.DB.Model(&user).Update("password", string(hashed))
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
