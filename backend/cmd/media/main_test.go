package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"omoikane-backend/internal/auth"
	"omoikane-backend/internal/database"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// makeRealPNG returns a small valid PNG so ServeMediaFile/thumbnail paths work.
func makeRealPNG() []byte {
	// 1x1 red PNG.
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x18, 0xdd, 0x8d, 0xb0,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func mediaTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// stubProducer records published CloudEvents without touching Kafka.
type stubProducer struct {
	published []events.CloudEvent
}

func (s *stubProducer) Close() error { return nil }

// setupMediaService boots the real media-service mux (newMediaMux) against the
// test database with a stub-producer relay, mirroring production.
func setupMediaService(t *testing.T) (*gorm.DB, *httptest.Server, *events.Relay, *stubProducer) {
	t.Helper()
	db, err := database.Connect(mediaTestDSN())
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
		UploadDir: t.TempDir(),
		Outbox:    outbox,
	}
	s := httptest.NewServer(newMediaMux(h))
	t.Cleanup(s.Close)

	stub := &stubProducer{}
	relay := events.NewRelay(outbox, stub, events.DefaultConfig())
	return db, s, relay, stub
}

func (s *stubProducer) Publish(_ context.Context, ev events.CloudEvent) error {
	s.published = append(s.published, ev)
	return nil
}

func TestMediaService_Health(t *testing.T) {
	_, s, _, _ := setupMediaService(t)

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

// TestMediaService_UploadToRelayPublish exercises the Phase 30 pipeline:
// authenticated upload -> transactional outbox row -> relay RunOnce -> stub
// producer -> row marked sent.
func TestMediaService_UploadToRelayPublish(t *testing.T) {
	db, s, relay, stub := setupMediaService(t)

	admin := models.User{Name: "Admin", Email: "admin@test.com", Password: "x", Role: "admin", Status: "active"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, err := auth.GenerateToken(admin.ID, "admin", "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	cookie := &http.Cookie{Name: "session", Value: token}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "pic.png")
	fw.Write(makeRealPNG())
	w.Close()

	req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d (%s)", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	var pending []events.OutboxEvent
	if err := db.Where("status = ?", events.OutboxPending).Find(&pending).Error; err != nil {
		t.Fatalf("query pending outbox: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(pending))
	}
	if pending[0].EventType != events.TypeMediaUploaded || pending[0].Subject != "media/1" {
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
	if stub.published[0].Type != events.TypeMediaUploaded || stub.published[0].Source != events.SourceMedia {
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

// TestMediaService_FileServingPublic: /media/file/{filename} is public (no
// auth); an unknown file is a 404, not a 401.
func TestMediaService_FileServingPublic(t *testing.T) {
	_, s, _, _ := setupMediaService(t)

	resp, err := http.Get(s.URL + "/media/file/definitely-missing.png")
	if err != nil {
		t.Fatalf("file request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for missing file, got %d", resp.StatusCode)
	}
}

// TestMediaService_ProtectedRoutesRejectAnonymous ensures every media-owned
// CRUD route is gated behind the middleware (guards the gateway flip).
func TestMediaService_ProtectedRoutesRejectAnonymous(t *testing.T) {
	_, s, _, _ := setupMediaService(t)

	for _, tc := range []struct {
		method, path string
	}{
		{"GET", "/media"},
		{"POST", "/media"},
		{"GET", "/media/1"},
		{"PUT", "/media/1"},
		{"DELETE", "/media/1"},
		{"POST", "/media/batch"},
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
