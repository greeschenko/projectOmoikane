package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"omoikane-backend/internal/auth"
	"omoikane-backend/internal/audit"
	"omoikane-backend/internal/mailer"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"
	"omoikane-backend/internal/recaptcha"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	RecaptchaToken string `json:"recaptchaToken,omitempty"`
}

type setupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type forgotPasswordRequest struct {
	Email          string `json:"email"`
	RecaptchaToken string `json:"recaptchaToken,omitempty"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// SetupStatus reports whether the site still needs initial admin setup.
// @Summary Check setup status
// @Description Returns whether an admin account still needs to be created (first run).
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /setup/check [get]
func (h *Handler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var count int64
	h.DB.Model(&models.User{}).Count(&count)

	json.NewEncoder(w).Encode(map[string]bool{"setupRequired": count == 0})
}

// Setup creates the initial admin account (only allowed before any user exists).
// @Summary Initialize admin account
// @Description Creates the first admin user. Only succeeds while the database has no users.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body setupRequest true "Email and password for the admin account"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /setup [post]
func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var count int64
	h.DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Setup already completed"})
		return
	}

	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email and password required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Name:     "Admin",
		Email:    req.Email,
		Password: string(hashed),
		Role:     "admin",
		Status:   "active",
	}
	if err := h.DB.Create(&user).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create admin"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Login authenticates a user and sets the session cookie.
// @Summary User login
// @Description Authenticates credentials, sets an HttpOnly session cookie and emits a login audit event.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "Email and password"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email and password required"})
		return
	}

	var user models.User
	if err := h.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		return
	}

	if user.Status == "banned" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Account is banned"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Role, h.JWTSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token generation failed"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     user.ID,
		UserName:   user.Name,
		Action:     "login",
		EntityType: "user",
		EntityID:   user.ID,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"role":    user.Role,
	})
}

// Register creates a new public user account.
// @Summary Register a user
// @Description Creates a new user account. Requires a valid reCAPTCHA token when reCAPTCHA is configured.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body registerRequest true "User details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "All fields required"})
		return
	}

	if h.RecaptchaSecret != "" {
		ok, err := recaptcha.VerifyToken(h.RecaptchaSecret, req.RecaptchaToken)
		if err != nil || !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "reCAPTCHA verification failed"})
			return
		}
	}

	var count int64
	h.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email already registered"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashed),
		Role:     "user",
		Status:   "active",
	}
	if err := h.DB.Create(&user).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    sanitizeUserJSON(user),
	})
}

// Logout clears the session cookie.
// @Summary User logout
// @Description Clears the session cookie and emits a logout audit event.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]bool
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := middleware.GetUserID(r)
	if userID > 0 {
		var user models.User
		h.DB.First(&user, userID)
		audit.Emit(h.AuditServiceURL, audit.Event{
			UserID:     userID,
			UserName:   user.Name,
			Action:     "logout",
			EntityType: "user",
			EntityID:   userID,
		})
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   0,
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ForgotPassword emails a password reset link (rate limited).
// @Summary Request password reset
// @Description Emails a password reset link if the account exists. Rate limited to 3 requests per 15 minutes per IP.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body forgotPasswordRequest true "Account email"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /auth/forgot-password [post]
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email required"})
		return
	}

	if h.RecaptchaSecret != "" {
		ok, err := recaptcha.VerifyToken(h.RecaptchaSecret, req.RecaptchaToken)
		if err != nil || !ok {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "reCAPTCHA verification failed"})
			return
		}
	}

	var user models.User
	err := h.DB.Where("email = ?", req.Email).First(&user).Error

	if err == nil {
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err == nil {
			tokenHex := hex.EncodeToString(tokenBytes)
			resetToken := models.PasswordResetToken{
				UserID:    user.ID,
				Token:     tokenHex,
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}
			h.DB.Create(&resetToken)

			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			frontendURL := scheme + "://" + r.Host
			resetLink := frontendURL + "/reset-password?token=" + tokenHex
			mailerCfg := mailer.Config{
				Host: h.SMTPHost,
				Port: h.SMTPPort,
				User: h.SMTPUser,
				Pass: h.SMTPPass,
				From: h.SMTPFrom,
			}

			settings := models.SiteSetting{}
			h.DB.First(&settings, 1)
			subject := settings.ResetEmailSubject
			if subject == "" {
				subject = "Password Reset Request"
			}
			bodyTemplate := settings.ResetEmailBodyHTML
			if bodyTemplate == "" {
				bodyTemplate = `<h2>Password Reset</h2><p>Click <a href="{{.ResetLink}}">here</a> to reset your password. Expires in {{.ExpiryHours}} hour(s).</p><p>If you did not request this, ignore this email.</p>`
			}

			bodyHTML, renderErr := mailer.RenderResetTemplate(bodyTemplate, mailer.ResetEmailData{
				ResetLink:   resetLink,
				SiteName:    settings.SiteName,
				ExpiryHours: 1,
			})
			if renderErr == nil {
				mailer.SendEmail(mailerCfg, user.Email, subject, bodyHTML)
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Check your email for a reset link",
	})
}

// ResetPassword sets a new password using a valid reset token.
// @Summary Reset password
// @Description Sets a new password using a token from the password reset email.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body resetPasswordRequest true "Reset token and new password"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /auth/reset-password [post]
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Token == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token and password required"})
		return
	}

	if errStr := validatePassword(req.Password); errStr != "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": errStr})
		return
	}

	var resetToken models.PasswordResetToken
	if err := h.DB.Where("token = ? AND used = ? AND expires_at > ?", req.Token, false, time.Now()).First(&resetToken).Error; err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired reset token"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to hash password"})
		return
	}

	if err := h.DB.Model(&models.User{}).Where("id = ?", resetToken.UserID).Update("password", string(hashed)).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update password"})
		return
	}

	h.DB.Model(&resetToken).Update("used", true)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
	})
}

// GetProfile returns the currently authenticated user's profile.
// @Summary Get current user profile
// @Description Returns id, name, email, role and avatar for the authenticated user.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /settings/profile [get]
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := middleware.GetUserID(r)
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     user.ID,
		"name":   user.Name,
		"email":  user.Email,
		"role":   user.Role,
		"avatar": user.Avatar,
	})
}

func sanitizeUserJSON(user models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"avatar":     user.Avatar,
		"createdAt":  user.CreatedAt,
		"updatedAt":  user.UpdatedAt,
	}
}

func validateEmail(email string) bool {
	return strings.Contains(email, "@")
}

func validatePassword(password string) string {
	if len(password) < 6 {
		return "Password must be at least 6 characters"
	}
	return ""
}
