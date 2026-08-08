package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"omoikane-backend/internal/auth"
)

type contextKey string

const (
	UserIDKey contextKey = "userId"
	RoleKey   contextKey = "role"
)

// ParseJWT extracts JWT claims from the session cookie or Authorization header.
func ParseJWT(r *http.Request, jwtSecret string) *auth.Claims {
	return extractClaims(r, jwtSecret)
}

// AuthRequired authenticates a request via JWT (cookie or bearer header).
func AuthRequired(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := extractClaims(r, jwtSecret)
		if claims == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		next(w, r.WithContext(ctx))
	}
}

// Authenticate resolves the identity for a request using either a JWT (cookie or
// bearer header) or, failing that, a stored API token presented as "Bearer <token>".
// Returns the resolved user ID and role, or false when no valid credential exists.
func Authenticate(r *http.Request, jwtSecret string, tokenStore TokenStore) (uint, string, bool) {
	if claims := extractClaims(r, jwtSecret); claims != nil {
		return claims.UserID, claims.Role, true
	}
	tokenString := bearerToken(r)
	if tokenString == "" || tokenStore == nil {
		return 0, "", false
	}
	return tokenStore.LookupToken(tokenString)
}

// TokenStore resolves an API token string to a user ID and role.
type TokenStore interface {
	LookupToken(raw string) (uint, string, bool)
}

func AdminRequired(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return AuthRequired(jwtSecret, func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(RoleKey).(string)
		if role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden"})
			return
		}
		next(w, r)
	})
}

func extractClaims(r *http.Request, jwtSecret string) *auth.Claims {
	tokenString := bearerToken(r)
	if tokenString == "" {
		return nil
	}

	claims, err := auth.ValidateToken(tokenString, jwtSecret)
	if err != nil {
		return nil
	}
	return claims
}

func bearerToken(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	ah := r.Header.Get("Authorization")
	if strings.HasPrefix(ah, "Bearer ") {
		return strings.TrimPrefix(ah, "Bearer ")
	}
	return ""
}

func GetUserID(r *http.Request) uint {
	if v, ok := r.Context().Value(UserIDKey).(uint); ok {
		return v
	}
	return 0
}

func GetRole(r *http.Request) string {
	if v, ok := r.Context().Value(RoleKey).(string); ok {
		return v
	}
	return ""
}

func WithUserID(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, RoleKey, role)
}
