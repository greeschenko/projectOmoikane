package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"omoikane-backend/internal/cache"
)

// CacheRead wraps a handler and caches successful (200) GET responses. It only
// caches requests without a session cookie or Authorization header, so
// authenticated responses (e.g. auth-aware page lists with drafts) never leak
// through the cache. Set to be used on public GET endpoints.
func CacheRead(c cache.Cache, ttl time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c == nil || !isPublicRequest(r) {
			next(w, r)
			return
		}

		key := "c:" + r.URL.RequestURI()
		if cached, ok := c.Get(r.Context(), key); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "hit")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, cached)
			return
		}

		w.Header().Set("X-Cache", "miss")
		cw := &cacheWriter{ResponseWriter: w, status: http.StatusOK}
		next(cw, r)
		if cw.status == http.StatusOK {
			c.Set(r.Context(), key, cw.body.String(), ttl)
		}
	}
}

// isPublicRequest reports whether the request carries no authentication,
// meaning its response is safe to share via the public cache.
func isPublicRequest(r *http.Request) bool {
	if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
		return false
	}
	if r.Header.Get("Authorization") != "" {
		return false
	}
	return true
}

type cacheWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (c *cacheWriter) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *cacheWriter) Write(b []byte) (int, error) {
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}
