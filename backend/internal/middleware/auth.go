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
	var tokenString string

	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		tokenString = cookie.Value
	}

	if tokenString == "" {
		ah := r.Header.Get("Authorization")
		if strings.HasPrefix(ah, "Bearer ") {
			tokenString = strings.TrimPrefix(ah, "Bearer ")
		}
	}

	if tokenString == "" {
		return nil
	}

	claims, err := auth.ValidateToken(tokenString, jwtSecret)
	if err != nil {
		return nil
	}
	return claims
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
