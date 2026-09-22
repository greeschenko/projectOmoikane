package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// InternalAuth guards service-to-service endpoints (Phase 31 aggregation). The
// caller must present the platform's shared internal token in the
// X-Internal-Token header. These endpoints are only reachable inside the
// compose network via the owning services' /internal/* prefixes; the gateway
// never exposes them.
func InternalAuth(expected string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Internal-Token")
		if expected == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}
		next(w, r)
	}
}
