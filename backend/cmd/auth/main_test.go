package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"omoikane-backend/internal/database"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// stubProducer records published CloudEvents without touching Kafka, so the
// full register -> outbox -> relay -> publish loop can be tested offline.
type stubProducer struct {
	published []events.CloudEvent
}

func (s *stubProducer) Publish(_ context.Context, ev events.CloudEvent) error {
	s.published = append(s.published, ev)
	return nil
}

func (s *stubProducer) Close() error { return nil }

func authTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// setupAuthService boots the real auth-service mux (newAuthMux) against the
// test database with a stub-producer relay, mirroring the production wiring.
func setupAuthService(t *testing.T) (*gorm.DB, *httptest.Server, *events.Relay, *stubProducer) {
	t.Helper()
	db, err := database.Connect(authTestDSN())
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
		// Auth owns the user trash entity (Phase 31); internal endpoints serve
		// it to the trash aggregator + dashboard facade.
		TrashEntities: []string{"user"},
	}
	s := httptest.NewServer(newAuthMux(h, "test-internal-token"))
	t.Cleanup(s.Close)

	stub := &stubProducer{}
	relay := events.NewRelay(outbox, stub, events.DefaultConfig())
	return db, s, relay, stub
}

func TestAuthService_Health(t *testing.T) {
	_, s, _, _ := setupAuthService(t)

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

func TestAuthService_SetupCheckPublic(t *testing.T) {
	_, s, _, _ := setupAuthService(t)

	resp, err := http.Get(s.URL + "/setup/check")
	if err != nil {
		t.Fatalf("setup check request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /setup/check, got %d", resp.StatusCode)
	}
}

// TestAuthService_RegisterToRelayPublish exercises the full Phase 29 pipeline:
// HTTP register -> transactional outbox row -> relay RunOnce -> stub producer
// -> row marked sent. Offline (no Kafka, no broker).
func TestAuthService_RegisterToRelayPublish(t *testing.T) {
	db, s, relay, stub := setupAuthService(t)

	resp, err := http.Post(s.URL+"/auth/register", "application/json",
		strings.NewReader(`{"name":"Service User","email":"svc@test.com","password":"SecurePass123!"}`))
	if err != nil {
		t.Fatalf("register request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Register must have enqueued exactly one pending user.registered row.
	var pending []events.OutboxEvent
	if err := db.Where("status = ?", events.OutboxPending).Find(&pending).Error; err != nil {
		t.Fatalf("query pending outbox: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(pending))
	}
	if pending[0].EventType != events.TypeUserRegistered || pending[0].Subject != "user/1" {
		t.Errorf("unexpected row: type=%s subject=%s", pending[0].EventType, pending[0].Subject)
	}

	// Relay flush (one batch) must publish via the stub producer and mark sent.
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
	if stub.published[0].Type != events.TypeUserRegistered || stub.published[0].Source != events.SourceAuth {
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

// TestAuthService_ProtectedRoutesRejectAnonymous ensures every auth-owned route
// that requires a session is gated behind the middleware (guards the gateway
// flip: previously the monolith enforced these; the service must too).
func TestAuthService_ProtectedRoutesRejectAnonymous(t *testing.T) {
	_, s, _, _ := setupAuthService(t)

	for _, tc := range []struct {
		method, path string
	}{
		{"GET", "/users"},
		{"POST", "/users"},
		{"GET", "/api-tokens"},
		{"POST", "/api-tokens"},
		{"GET", "/settings/profile"},
		{"PUT", "/settings/profile"},
		{"POST", "/settings/password"},
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

// TestAuthService_InternalEndpoints covers the Phase 31 /internal/* surface:
// it must be token-gated (401 without the shared internal token) and serve the
// dashboard facade (stats) + trash aggregator (user entity) within the rows the
// auth service owns.
func TestAuthService_InternalEndpoints(t *testing.T) {
	db, s, _, _ := setupAuthService(t)

	// 401 without the shared internal token.
	for _, path := range []string{"/internal/stats", "/internal/trash", "/internal/trash/count"} {
		resp, err := http.Get(s.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s without token: expected 401, got %d", path, resp.StatusCode)
		}
	}

	internal := func(method, path string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(method, s.URL+path, nil)
		req.Header.Set("X-Internal-Token", "test-internal-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return resp
	}

	// Stats: user count + 7-day zero-filled registrations chart.
	u := models.User{Name: "Internal", Email: "i@t.com", Password: "x", Role: "admin", Status: "active"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	resp := internal("GET", "/internal/stats")
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("GET /internal/stats: expected 200, got %d", resp.StatusCode)
	}
	var stats struct {
		Users               int64                    `json:"users"`
		RecentRegistrations []map[string]interface{} `json:"recentRegistrations"`
	}
	json.NewDecoder(resp.Body).Decode(&stats)
	resp.Body.Close()
	if stats.Users != 1 || len(stats.RecentRegistrations) != 7 {
		t.Errorf("stats = %+v, want users=1, 7 registration entries", stats)
	}

	// Trash: soft-delete a user; list + count must show it; restore clears it.
	db.Delete(&u)
	resp = internal("GET", "/internal/trash/count")
	var count struct {
		Count int64 `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&count)
	resp.Body.Close()
	if count.Count != 1 {
		t.Errorf("trash count = %d, want 1", count.Count)
	}
	resp = internal("GET", "/internal/trash")
	var items []json.RawMessage
	json.NewDecoder(resp.Body).Decode(&items)
	resp.Body.Close()
	if len(items) != 1 {
		t.Errorf("trash list = %d items, want 1", len(items))
	}
	resp = internal("POST", fmt.Sprintf("/internal/trash/user/%d/restore", u.ID))
	code := resp.StatusCode
	resp.Body.Close()
	if code != http.StatusOK {
		t.Errorf("restore: expected 200, got %d", code)
	}
	resp = internal("GET", "/internal/trash/count")
	json.NewDecoder(resp.Body).Decode(&count)
	resp.Body.Close()
	if count.Count != 0 {
		t.Errorf("trash count after restore = %d, want 0", count.Count)
	}
}
