package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"omoikane-backend/internal/cache"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/middleware"

	"gorm.io/gorm"
)

func cacheServer(db *gorm.DB) (*httptest.Server, cache.Cache) {
	mr := miniredis.NewMiniRedis()
	_ = mr.Start()
	c, err := cache.NewRedis("redis://"+mr.Addr(), time.Minute)
	if err != nil {
		panic(err)
	}
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret, Cache: c}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /pages", middleware.CacheRead(c, time.Minute, h.GetPages))
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	return httptest.NewServer(mux), c
}

func TestCache_WriteFlushesPublicRead(t *testing.T) {
	db := setupTestDB(t)
	server, _ := cacheServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "admin@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "admin@test.com", "pass123")

	// First public read populates the cache
	resp1 := unauthenticatedGet(t, server.URL+"/pages")
	body1 := readBody(t, resp1)
	if resp1.Header.Get("X-Cache") != "miss" {
		t.Fatalf("Expected first read miss, got %q", resp1.Header.Get("X-Cache"))
	}

	// Second public read should be served from cache
	resp2 := unauthenticatedGet(t, server.URL+"/pages")
	body2 := readBody(t, resp2)
	if resp2.Header.Get("X-Cache") != "hit" {
		t.Fatalf("Expected second read hit, got %q", resp2.Header.Get("X-Cache"))
	}
	if body2 != body1 {
		t.Fatal("Cached body mismatch")
	}

	// A write (create page) must flush the cache
	createResp := authenticatedRequest(t, "POST", server.URL+"/pages",
		`{"title":"Cached","slug":"cached","content":"hi","status":"published"}`, cookie)
	if createResp.StatusCode != 201 {
		t.Fatalf("Create page failed (status %d): %s", createResp.StatusCode, readBody(t, createResp))
	}
	createResp.Body.Close()

	// Next public read must be a fresh miss containing the new page
	resp3 := unauthenticatedGet(t, server.URL+"/pages")
	body3 := readBody(t, resp3)
	if resp3.Header.Get("X-Cache") != "miss" {
		t.Fatalf("Expected miss after write, got %q", resp3.Header.Get("X-Cache"))
	}
	if !strings.Contains(body3, "cached") {
		t.Fatalf("Fresh read missing new page: %s", body3)
	}
}

func unauthenticatedGet(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}
