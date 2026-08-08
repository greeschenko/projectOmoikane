package handlers

import (
	"encoding/json"
	"net/http"

	"omoikane-backend/internal/audit"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type updateSettingsRequest struct {
	SiteName           *string `json:"siteName,omitempty"`
	Tagline            *string `json:"tagline,omitempty"`
	Logo               *string `json:"logo,omitempty"`
	Favicon            *string `json:"favicon,omitempty"`
	BlogEnabled        *bool   `json:"blogEnabled,omitempty"`
	ResetEmailSubject  *string `json:"resetEmailSubject,omitempty"`
	ResetEmailBodyHTML *string `json:"resetEmailBodyHTML,omitempty"`
}

type updateProfileRequest struct {
	Name   *string `json:"name,omitempty"`
	Email  *string `json:"email,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// GetSettings returns public site settings.
// @Summary Get site settings
// @Description Returns site name, tagline, logo, favicon, blog toggle and email template settings.
// @Tags settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	settings := models.SiteSetting{}
	result := h.DB.First(&settings, 1)
	if result.Error != nil {
		settings = models.SiteSetting{
			SiteName:           "Omoikane",
			Tagline:            "A headless-ish CMS",
			BlogEnabled:        true,
			ResetEmailSubject:  "Password Reset Request",
			ResetEmailBodyHTML: `<h2>Password Reset</h2><p>Click <a href="{{.ResetLink}}">here</a> to reset your password. Expires in {{.ExpiryHours}} hour(s).</p><p>If you did not request this, ignore this email.</p>`,
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

// UpdateSettings updates site settings (admin only).
// @Summary Update site settings
// @Description Updates the provided site settings fields.
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body updateSettingsRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /settings [put]
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
			SiteName:           "Omoikane",
			Tagline:            "A headless-ish CMS",
			BlogEnabled:        true,
			ResetEmailSubject:  "Password Reset Request",
			ResetEmailBodyHTML: `<h2>Password Reset</h2><p>Click <a href="{{.ResetLink}}">here</a> to reset your password. Expires in {{.ExpiryHours}} hour(s).</p><p>If you did not request this, ignore this email.</p>`,
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

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "update",
		EntityType: "settings",
		Detail:     "Updated site settings",
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"siteName":          settings.SiteName,
		"tagline":           settings.Tagline,
		"logo":              settings.Logo,
		"favicon":           settings.Favicon,
		"blogEnabled":       settings.BlogEnabled,
		"resetEmailSubject": settings.ResetEmailSubject,
		"resetEmailBodyHTML": settings.ResetEmailBodyHTML,
	})
	h.flushCache()
}

// UpdateProfile updates the current user's profile.
// @Summary Update profile
// @Description Updates the current user's name, email and/or avatar.
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body updateProfileRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /settings/profile [put]
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

// ChangePassword changes the current user's password.
// @Summary Change password
// @Description Changes the current user's password after verifying the current one.
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body changePasswordRequest true "Current and new password"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /settings/password [post]
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
