package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"omoikane-backend/internal/auth"
	"omoikane-backend/internal/database"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// stubProducer records published CloudEvents without touching Kafka.
type stubProducer struct {
	published []events.CloudEvent
}

func (s *stubProducer) Publish(_ context.Context, ev events.CloudEvent) error {
	s.published = append(s.published, ev)
	return nil
}

func (s *stubProducer) Close() error { return nil }

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func messagesTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// setupMessagesService boots the real messages-service mux (newMessagesMux)
// against the test database with the outbox wired, mirroring production.
func setupMessagesService(t *testing.T) (*gorm.DB, *httptest.Server) {
	t.Helper()
	db, err := database.Connect(messagesTestDSN())
	if err != nil {
		t.Fatalf("connect test DB: %v", err)
	}
	if sqlDB, serr := db.DB(); serr == nil {
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(3)
	}
	t.Cleanup(func() {
		if sqlDB, serr := db.DB(); serr == nil {
			sqlDB.Close()
		}
	})
	if err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE").Error; err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test DB: %v", err)
	}
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}

	h := &handlers.Handler{
		DB:        db,
		JWTSecret: "test-secret",
		// Messages owns the contact + message trash entities (Phase 31).
		TrashEntities: []string{"contact", "message"},
		Outbox:        events.NewGormOutboxStore(db),
	}
	s := httptest.NewServer(newMessagesMux(h, "test-internal-token"))
	t.Cleanup(s.Close)
	return db, s
}

// sessionCookie mints a JWT for a user and returns it as the "session" cookie
// the middleware expects. The messages mux has no /auth/login route.
func sessionCookie(t *testing.T, userID uint, role string) map[string][]string {
	t.Helper()
	token, err := auth.GenerateToken(userID, role, "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	return map[string][]string{"Cookie": {"session=" + token}}
}

func TestMessagesService_Health(t *testing.T) {
	_, s := setupMessagesService(t)

	resp, err := http.Get(s.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status = %q, want ok", body["status"])
	}
}

// TestMessagesService_ContactPublic: the contact form must be POSTable without
// any credentials (gateway flips /api/contact to this service). ReCAPTCHA is
// disabled in tests (RecaptchaSecret empty), mirroring the relaxed dev config.
func TestMessagesService_ContactPublic(t *testing.T) {
	db, s := setupMessagesService(t)

	resp, err := http.Post(s.URL+"/contact", "application/json",
		strings.NewReader(`{"name":"Jane","email":"jane@test.com","subject":"Hello","message":"Body text"}`))
	if err != nil {
		t.Fatalf("contact request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 from POST /contact, got %d (%s)", resp.StatusCode, readBody(resp))
	}
	resp.Body.Close()

	var count int64
	db.Model(&models.ContactMessage{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 contact row, got %d", count)
	}
}

// TestMessagesService_InternalStats exercises the dashboard facade feed:
// the shared internal token is required (401 without it) and the payload
// carries the exact Phase 26 recent-messages shape.
func TestMessagesService_InternalStats(t *testing.T) {
	db, s := setupMessagesService(t)
	db.Create(&models.Message{Title: "One", Content: "a"})
	db.Create(&models.Message{Title: "Two", Content: "b"})

	resp, err := http.Get(s.URL + "/internal/stats")
	if err != nil {
		t.Fatalf("stats request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("stats without token: expected 401, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", s.URL+"/internal/stats", nil)
	req.Header.Set("X-Internal-Token", "test-internal-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stats request with token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /internal/stats, got %d", resp.StatusCode)
	}
	var stats struct {
		Messages       int64 `json:"messages"`
		RecentMessages []struct {
			ID    uint   `json:"id"`
			Title string `json:"title"`
		} `json:"recentMessages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Messages != 2 || len(stats.RecentMessages) != 2 {
		t.Errorf("stats = %+v, want messages=2 recentMessages=2", stats)
	}
}

// TestMessagesService_InternalTrashScoped: the internal trash surface only
// serves the entities this service owns (contact + message); others 400.
func TestMessagesService_InternalTrashScoped(t *testing.T) {
	db, s := setupMessagesService(t)

	cm := models.ContactMessage{Name: "Jane", Email: "j@test.com", Subject: "Hi", Message: "m"}
	db.Create(&cm)
	m := models.Message{Title: "Broadcast", Content: "body"}
	db.Create(&m)
	db.Delete(&cm)
	db.Delete(&m)

	do := func(method, path string, want int) string {
		t.Helper()
		req, _ := http.NewRequest(method, s.URL+path, nil)
		req.Header.Set("X-Internal-Token", "test-internal-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		body := readBody(resp)
		if resp.StatusCode != want {
			t.Errorf("%s %s: expected %d, got %d (%s)", method, path, want, resp.StatusCode, body)
		}
		return body
	}

	do("GET", "/internal/trash", http.StatusOK)
	body := do("GET", "/internal/trash/count", http.StatusOK)
	var out struct {
		Count int64 `json:"count"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if out.Count != 2 {
		t.Errorf("internal trash count = %d, want 2 (contact + message)", out.Count)
	}

	// A page row is NOT owned by messages -> 400, never proxied by the trash
	// service (it would route page to content-service instead).
	do("POST", "/internal/trash/page/1/restore", http.StatusBadRequest)

	// Restore the soft-deleted message row.
	do("POST", "/internal/trash/message/"+formatID(m.ID)+"/restore", http.StatusOK)
	body = do("GET", "/internal/trash/count", http.StatusOK)
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if out.Count != 1 {
		t.Errorf("internal trash count after restore = %d, want 1", out.Count)
	}

	// Hard-delete the contact row for good.
	do("DELETE", "/internal/trash/contact/"+formatID(cm.ID), http.StatusOK)
	body = do("GET", "/internal/trash/count", http.StatusOK)
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if out.Count != 0 {
		t.Errorf("internal trash count after hard-delete = %d, want 0", out.Count)
	}
}

// TestMessagesService_ProtectedRoutesRejectAnonymous guards the gateway flip:
// every messages/contacts route is gated behind Auth/Admin middleware.
func TestMessagesService_ProtectedRoutesRejectAnonymous(t *testing.T) {
	_, s := setupMessagesService(t)

	for _, tc := range []struct {
		method, path string
	}{
		{"GET", "/messages"},
		{"GET", "/messages/1"},
		{"POST", "/messages/1/read"},
		{"POST", "/messages/read-all"},
		{"POST", "/messages"},
		{"DELETE", "/messages"},
		{"DELETE", "/messages/1"},
		{"GET", "/contacts"},
		{"GET", "/contacts/1"},
		{"POST", "/contacts/1/read"},
		{"DELETE", "/contacts/1"},
	} {
		req, err := http.NewRequest(tc.method, s.URL+tc.path, nil)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}
}

// TestMessagesService_RoleGating: reads are authed, writes are admin-only.
func TestMessagesService_RoleGating(t *testing.T) {
	db, s := setupMessagesService(t)
	user := models.User{Name: "U", Email: "u@t.com", Password: "x", Role: "user", Status: "active"}
	admin := models.User{Name: "A", Email: "a@t.com", Password: "x", Role: "admin", Status: "active"}
	db.Create(&user)
	db.Create(&admin)

	req, _ := http.NewRequest("GET", s.URL+"/messages", nil)
	for k, v := range sessionCookie(t, user.ID, "user") {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("user GET /messages: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("user GET /messages: expected 200, got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest("POST", s.URL+"/messages",
		strings.NewReader(`{"title":"T","content":"c"}`))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range sessionCookie(t, user.ID, "user") {
		req.Header[k] = v
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("user POST /messages: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("user POST /messages: expected 403, got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest("POST", s.URL+"/messages",
		strings.NewReader(`{"title":"T","content":"c"}`))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range sessionCookie(t, admin.ID, "admin") {
		req.Header[k] = v
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("admin POST /messages: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Errorf("admin POST /messages: expected 200/201, got %d", resp.StatusCode)
	}
}

func formatID(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
