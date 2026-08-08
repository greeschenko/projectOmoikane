package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"omoikane-backend/internal/cache"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

type Handler struct {
	DB              *gorm.DB
	JWTSecret       string
	UploadDir       string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	RecaptchaSecret string
	AuditServiceURL string
	Cache           cache.Cache
}

// flushCache invalidates the response cache after any write to a cached entity.
func (h *Handler) flushCache() {
	if h.Cache != nil {
		h.Cache.Flush(context.Background())
	}
}

// LookupToken resolves a raw API token to (userID, role, ok). Used by the
// middleware to authenticate Bearer tokens presented in the Authorization header.
func (h *Handler) LookupToken(raw string) (uint, string, bool) {
	if h.DB == nil {
		return 0, "", false
	}
	hash := sha256.Sum256([]byte(raw))
	var token models.ApiToken
	if err := h.DB.Where("token_hash = ?", hex.EncodeToString(hash[:])).First(&token).Error; err != nil {
		return 0, "", false
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return 0, "", false
	}
	now := time.Now()
	h.DB.Model(&token).Update("last_used_at", now)
	return token.ID, token.Role, true
}

func (h *Handler) Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, role, ok := middleware.Authenticate(r, h.JWTSecret, h)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		ctx := middleware.WithUserID(r.Context(), userID)
		ctx = middleware.WithRole(ctx, role)
		next(w, r.WithContext(ctx))
	}
}

func (h *Handler) Admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, role, ok := middleware.Authenticate(r, h.JWTSecret, h)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		if role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			http.Error(w, "", http.StatusForbidden)
			return
		}
		ctx := middleware.WithUserID(r.Context(), userID)
		ctx = middleware.WithRole(ctx, role)
		next(w, r.WithContext(ctx))
	}
}

// batchRequest is the shared body for batch actions across entities.
type batchRequest struct {
	Action string `json:"action"` // "delete", "publish", "draft", "ban", "activate", "clear"
	IDs    []uint `json:"ids"`
}
