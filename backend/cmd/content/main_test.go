package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func contentTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// setupContentService boots the real content-service mux (newContentMux)
// against the test database with a stub-producer relay, mirroring production.
func setupContentService(t *testing.T) (*gorm.DB, *httptest.Server, *events.Relay, *stubProducer) {
	t.Helper()
	db, err := database.Connect(contentTestDSN())
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

	outbox := events.NewGormOutboxStore(db)
	h := &handlers.Handler{
		DB:        db,
		JWTSecret: "test-secret",
		Outbox:    outbox,
	}
	s := httptest.NewServer(newContentMux(h))
	t.Cleanup(s.Close)

	stub := &stubProducer{}
	relay := events.NewRelay(outbox, stub, events.DefaultConfig())
	return db, s, relay, stub
}

// sessionCookie mints an admin JWT for a user and returns it as the "session"
// cookie the middleware expects. The content mux has no /auth/login route.
func sessionCookie(t *testing.T, userID uint, role string) *http.Cookie {
	t.Helper()
	token, err := auth.GenerateToken(userID, role, "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	return &http.Cookie{Name: "session", Value: token}
}

func TestContentService_Health(t *testing.T) {
	_, s, _, _ := setupContentService(t)

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

// TestContentService_PublicReadsUnauthed: pages + posts lists must be public
// (gateway flips /api/pages and /api/blog/ to this service; SSR + public reads
// hit them anonymously).
func TestContentService_PublicReadsUnauthed(t *testing.T) {
	_, s, _, _ := setupContentService(t)

	for _, path := range []string{"/pages", "/blog/posts"} {
		resp, err := http.Get(s.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
		}
	}
}

// TestContentService_PublishPageToRelayPublish exercises the Phase 30 pipeline:
// authenticated page create (published) -> transactional outbox row -> relay
// RunOnce -> stub producer -> row marked sent.
func TestContentService_PublishPageToRelayPublish(t *testing.T) {
	db, s, relay, stub := setupContentService(t)

	admin := models.User{Name: "Admin", Email: "admin@test.com", Password: "x", Role: "admin", Status: "active"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	cookie := sessionCookie(t, admin.ID, "admin")

	req, _ := http.NewRequest("POST", s.URL+"/pages",
		strings.NewReader(`{"title":"About","slug":"about","content":"<p>hi</p>","status":"published"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create page request: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected page create success, got %d (%s)", resp.StatusCode, readBody(resp))
	}
	resp.Body.Close()

	var pending []events.OutboxEvent
	if err := db.Where("status = ?", events.OutboxPending).Find(&pending).Error; err != nil {
		t.Fatalf("query pending outbox: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(pending))
	}
	if pending[0].EventType != events.TypePagePublished || pending[0].Subject != "page/1" {
		t.Errorf("unexpected row: type=%s subject=%s", pending[0].EventType, pending[0].Subject)
	}

	attempted, err := relay.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("relay RunOnce: %v", err)
	}
	if attempted != 1 {
		t.Errorf("relay attempted %d rows, want 1", attempted)
	}
	if len(stub.published) != 1 {
		t.Fatalf("stub producer published %d events, want 1", len(stub.published))
	}
	if stub.published[0].Type != events.TypePagePublished || stub.published[0].Source != events.SourceContent {
		t.Errorf("published event mismatch: %+v", stub.published[0])
	}

	var remaining []events.OutboxEvent
	if err := db.Where("status = ?", events.OutboxPending).Find(&remaining).Error; err != nil {
		t.Fatalf("query remaining outbox: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected no pending rows after relay, got %d", len(remaining))
	}
}

// TestContentService_PublishPostToRelayPublish covers the post.published path
// with tags: create a published post with pre-existing tags -> relay -> stub.
func TestContentService_PublishPostToRelayPublish(t *testing.T) {
	db, s, relay, stub := setupContentService(t)

	admin := models.User{Name: "Admin", Email: "admin@test.com", Password: "x", Role: "admin", Status: "active"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	db.Create(&models.Tag{Name: "go", Slug: "go"})
	db.Create(&models.Tag{Name: "events", Slug: "events"})
	cookie := sessionCookie(t, admin.ID, "admin")

	req, _ := http.NewRequest("POST", s.URL+"/blog/posts",
		strings.NewReader(`{"title":"Post","slug":"post-1","content":"<p>x</p>","status":"published","tags":["go","events"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create post request: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected post create success, got %d (%s)", resp.StatusCode, readBody(resp))
	}
	resp.Body.Close()

	var pending []events.OutboxEvent
	if err := db.Where("status = ?", events.OutboxPending).Find(&pending).Error; err != nil {
		t.Fatalf("query pending outbox: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(pending))
	}
	if pending[0].EventType != events.TypePostPublished || pending[0].Subject != "post/1" {
		t.Errorf("unexpected row: type=%s subject=%s", pending[0].EventType, pending[0].Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(pending[0].Payload))
	if err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	var data struct {
		TagIDs []uint `json:"tagIds"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if len(data.TagIDs) != 2 {
		t.Errorf("expected tagIds [1 2], got %v", data.TagIDs)
	}

	if _, err := relay.RunOnce(context.Background()); err != nil {
		t.Fatalf("relay RunOnce: %v", err)
	}
	if len(stub.published) != 1 || stub.published[0].Type != events.TypePostPublished {
		t.Errorf("stub published %d events, want 1 post.published", len(stub.published))
	}
}

// TestContentService_ProtectedRoutesRejectAnonymous ensures every content-owned
// write route is gated behind the middleware (guards the gateway flip).
func TestContentService_ProtectedRoutesRejectAnonymous(t *testing.T) {
	_, s, _, _ := setupContentService(t)

	for _, tc := range []struct {
		method, path string
	}{
		{"POST", "/pages"},
		{"PUT", "/pages/1"},
		{"DELETE", "/pages/1"},
		{"POST", "/pages/batch"},
		{"PUT", "/pages/reorder"},
		{"GET", "/admin/blog/posts"},
		{"POST", "/blog/posts"},
		{"PUT", "/blog/posts/1"},
		{"DELETE", "/blog/posts/1"},
		{"POST", "/blog/posts/batch"},
		{"POST", "/blog/posts/1/like"},
		{"POST", "/blog/tags"},
		{"DELETE", "/blog/tags/1"},
		{"POST", "/blog/categories"},
		{"DELETE", "/blog/categories/1"},
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
