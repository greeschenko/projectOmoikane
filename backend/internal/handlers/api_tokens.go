package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"omoikane-backend/internal/models"
)

type createApiTokenRequest struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	ExpiresIn   int    `json:"expiresInDays"` // 0 = never
	Description string `json:"description"`
}

// GetApiTokens lists API tokens (admin only).
// @Summary List API tokens
// @Description Returns all API tokens without their secrets (hashes only).
// @Tags api-tokens
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{}
// @Router /api-tokens [get]
func (h *Handler) GetApiTokens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var tokens []models.ApiToken
	h.DB.Order("created_at desc").Find(&tokens)

	result := make([]map[string]interface{}, 0)
	for _, t := range tokens {
		result = append(result, map[string]interface{}{
			"id":          t.ID,
			"name":        t.Name,
			"role":        t.Role,
			"description": t.Description,
			"expiresAt":   t.ExpiresAt,
			"lastUsedAt":  t.LastUsedAt,
			"createdAt":   t.CreatedAt,
		})
	}
	json.NewEncoder(w).Encode(result)
}

// CreateApiToken creates an API token (admin only). The raw token is returned once.
// @Summary Create API token
// @Description Creates a new API token. The raw token is returned only once; store the SHA-256 hash instead.
// @Tags api-tokens
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createApiTokenRequest true "Token name, role and optional expiry in days"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api-tokens [post]
func (h *Handler) CreateApiToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req createApiTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name required"})
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		req.Role = "admin"
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token generation failed"})
		return
	}
	rawToken := base64.RawURLEncoding.EncodeToString(raw)

	hash := sha256.Sum256([]byte(rawToken))
	token := models.ApiToken{
		Name:        req.Name,
		TokenHash:   hex.EncodeToString(hash[:]),
		Role:        req.Role,
		Description: req.Description,
	}
	if req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresIn) * 24 * time.Hour)
		token.ExpiresAt = &exp
	}
	if err := h.DB.Create(&token).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create token"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        token.ID,
		"token":     rawToken,
		"name":      token.Name,
		"role":      token.Role,
		"expiresAt": token.ExpiresAt,
	})
}

// DeleteApiToken revokes an API token (admin only).
// @Summary Revoke API token
// @Description Soft-deletes an API token so it can no longer authenticate.
// @Tags api-tokens
// @Produce json
// @Security BearerAuth
// @Param id path int true "Token ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /api-tokens/{id} [delete]
func (h *Handler) DeleteApiToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid token ID"})
		return
	}

	var token models.ApiToken
	if err := h.DB.First(&token, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token not found"})
		return
	}

	h.DB.Delete(&token)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
