package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"omoikane-backend/internal/cache"
)

func miniredisCache(t *testing.T) (cache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c, err := cache.NewRedis("redis://"+mr.Addr(), time.Minute)
	if err != nil {
		t.Fatalf("Failed to create redis cache: %v", err)
	}
	return c, mr
}

func writeJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": r.URL.RequestURI()})
}

func TestCacheRead_CachesSecondCall(t *testing.T) {
	c, _ := miniredisCache(t)
	handler := CacheRead(c, time.Minute, http.HandlerFunc(writeJSON))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest("GET", "/pages", nil))
	if first.Header().Get("X-Cache") != "miss" {
		t.Fatalf("Expected X-Cache miss, got %q", first.Header().Get("X-Cache"))
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest("GET", "/pages", nil))
	if second.Header().Get("X-Cache") != "hit" {
		t.Fatalf("Expected X-Cache hit, got %q", second.Header().Get("X-Cache"))
	}
	if second.Body.String() != first.Body.String() {
		t.Fatalf("Cached body mismatch: %q != %q", second.Body.String(), first.Body.String())
	}
}

func TestCacheRead_DistinguishesQueryStrings(t *testing.T) {
	c, _ := miniredisCache(t)
	handler := CacheRead(c, time.Minute, http.HandlerFunc(writeJSON))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/pages", nil))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/pages?menu=true", nil))

	if rec.Header().Get("X-Cache") != "miss" {
		t.Fatalf("Expected query-distinct miss, got %q", rec.Header().Get("X-Cache"))
	}
	if !strings.Contains(rec.Body.String(), "menu=true") {
		t.Fatalf("Response was served from the wrong cache key: %q", rec.Body.String())
	}
}

func TestCacheRead_BypassesAuthenticatedRequests(t *testing.T) {
	c, _ := miniredisCache(t)
	handler := CacheRead(c, time.Minute, http.HandlerFunc(writeJSON))

	req := httptest.NewRequest("GET", "/pages", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Cache") != "" {
		t.Fatalf("Authenticated request must bypass cache, got X-Cache %q", rec.Header().Get("X-Cache"))
	}

	// Public request now should NOT be served a cached (auth) response
	pub := httptest.NewRecorder()
	handler.ServeHTTP(pub, httptest.NewRequest("GET", "/pages", nil))
	if pub.Header().Get("X-Cache") != "miss" {
		t.Fatalf("Public request after auth request should be a miss, got %q", pub.Header().Get("X-Cache"))
	}
}

func TestCacheRead_BypassesBearerAuthorization(t *testing.T) {
	c, _ := miniredisCache(t)
	handler := CacheRead(c, time.Minute, http.HandlerFunc(writeJSON))

	req := httptest.NewRequest("GET", "/pages", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Cache") != "" {
		t.Fatalf("Bearer request must bypass cache, got X-Cache %q", rec.Header().Get("X-Cache"))
	}
}

func TestRedisCache_FlushInvalidates(t *testing.T) {
	c, _ := miniredisCache(t)
	handler := CacheRead(c, time.Minute, http.HandlerFunc(writeJSON))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/pages", nil))
	c.Flush(context.Background())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/pages", nil))
	if rec.Header().Get("X-Cache") != "miss" {
		t.Fatalf("Expected miss after flush, got %q", rec.Header().Get("X-Cache"))
	}
}

func TestRedisCache_RedisDownPassthrough(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer client.Close()
	c := cache.NewRedisClient(client)
	handler := CacheRead(c, time.Minute, http.HandlerFunc(writeJSON))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/pages", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected passthrough response, got %d", rec.Code)
	}
}
