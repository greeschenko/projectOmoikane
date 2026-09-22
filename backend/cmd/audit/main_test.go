package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"omoikane-backend/internal/auth"
	"omoikane-backend/internal/database"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// auditTestDSN points at the shared test database (same omoikane_test all
// cmd/* tests use; existing tables are dropped per test).
func auditTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// setupAuditService boots the real audit mux (newAuditMux) against a clean
// audit_logs table. The package-level db is assigned so handleGetLogs and the
// consumer handler write through the test connection (same as production
// wiring, where db is the audit database handle).
func setupAuditService(t *testing.T) *gorm.DB {
	t.Helper()
	testDB, err := database.Connect(auditTestDSN())
	if err != nil {
		t.Fatalf("connect test DB: %v", err)
	}
	if sqlDB, serr := testDB.DB(); serr == nil {
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(3)
	}
	t.Cleanup(func() {
		if sqlDB, serr := testDB.DB(); serr == nil {
			sqlDB.Close()
		}
	})
	if err := testDB.Exec("DROP SCHEMA IF EXISTS public CASCADE").Error; err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if err := testDB.Exec("CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if err := testDB.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("migrate audit_logs: %v", err)
	}
	db = testDB
	return testDB
}

// adminCookie mints an admin session cookie against the test JWT secret.
func adminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	return sessionCookie(t, 1, "admin")
}

func sessionCookie(t *testing.T, userID uint, role string) *http.Cookie {
	t.Helper()
	token, err := auth.GenerateToken(userID, role, "test-secret")
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return &http.Cookie{Name: "session", Value: token}
}

func TestAuditService_Health(t *testing.T) {
	db := setupAuditService(t)
	srv := httptest.NewServer(newAuditMux("test-secret"))
	defer srv.Close()
	_ = db

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d", resp.StatusCode)
	}
}

func TestAuditService_LogsAdminGate(t *testing.T) {
	db := setupAuditService(t)
	srv := httptest.NewServer(newAuditMux("test-secret"))
	defer srv.Close()
	_ = db

	// No cookie -> 401.
	resp, err := http.Get(srv.URL + "/audit-logs")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}

	// Non-admin cookie -> 403.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/audit-logs", nil)
	req.AddCookie(sessionCookie(t, 2, "user"))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", resp.StatusCode)
	}

	// Admin cookie -> 200.
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/audit-logs", nil)
	req.AddCookie(adminCookie(t))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d", resp.StatusCode)
	}
}

func TestAuditService_LogsListing(t *testing.T) {
	db := setupAuditService(t)
	srv := httptest.NewServer(newAuditMux("test-secret"))
	defer srv.Close()

	seeded := []models.AuditLog{
		{EventID: "e1", UserID: 1, UserName: "admin@test.com", Action: "publish", EntityType: "post", EntityID: 7, Detail: "Published post \"Hello\""},
		{EventID: "e2", UserID: 2, UserName: "jane@test.com", Action: "register", EntityType: "user", EntityID: 2, Detail: "Registered user jane@test.com"},
		{EventID: "e3", UserName: "system", Action: "upload", EntityType: "media", EntityID: 3, Detail: "Uploaded media \"logo.png\""},
	}
	for _, row := range seeded {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	// Entity filter on admin surface.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/audit-logs?entity=post", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var body struct {
		Logs  []models.AuditLog `json:"logs"`
		Total int64             `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 || len(body.Logs) != 1 {
		t.Fatalf("entity=post -> total=%d len=%d, want 1/1", body.Total, len(body.Logs))
	}
	if body.Logs[0].EntityType != "post" || body.Logs[0].Action != "publish" || body.Logs[0].EventID != "e1" {
		t.Errorf("unexpected row: %+v", body.Logs[0])
	}

	// Search filter on raw /logs.
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/logs?search=jane", nil)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp2.Body.Close()
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 {
		t.Errorf("search=jane total=%d, want 1", body.Total)
	}
}

// testCloudEvent builds a CloudEvent with marshaled payload data.
func testCloudEvent(typ string, data map[string]any) events.CloudEvent {
	raw, _ := json.Marshal(data)
	return events.CloudEvent{
		SpecVersion: "1.0",
		ID:          "ev-" + typ,
		Source:      "test",
		Type:        typ,
		Subject:     "test",
		Data:        raw,
	}
}

func TestMapEventToLog_CatalogTypes(t *testing.T) {
	cases := []struct {
		name       string
		ev         events.CloudEvent
		action     string
		entityType string
		entityID   uint
		userName   string
	}{
		{
			name:       "user.registered",
			ev:         testCloudEvent(events.TypeUserRegistered, map[string]any{"id": 3, "email": "a@b.c", "role": "editor"}),
			action:     "register",
			entityType: "user",
			entityID:   3,
			userName:   "a@b.c",
		},
		{
			name:       "auth.login",
			ev:         testCloudEvent(events.TypeAuthLogin, map[string]any{"id": 5, "email": "u@b.c", "method": "cookie"}),
			action:     "login",
			entityType: "user",
			entityID:   5,
			userName:   "u@b.c",
		},
		{
			name:       "page.published",
			ev:         testCloudEvent(events.TypePagePublished, map[string]any{"id": 11, "slug": "about", "title": "About Us"}),
			action:     "publish",
			entityType: "page",
			entityID:   11,
			userName:   "system",
		},
		{
			name:       "post.published",
			ev:         testCloudEvent(events.TypePostPublished, map[string]any{"id": 12, "slug": "hello", "title": "Hello"}),
			action:     "publish",
			entityType: "post",
			entityID:   12,
			userName:   "system",
		},
		{
			name:       "media.uploaded",
			ev:         testCloudEvent(events.TypeMediaUploaded, map[string]any{"id": 8, "filename": "logo.png", "size": 1024}),
			action:     "upload",
			entityType: "media",
			entityID:   8,
			userName:   "system",
		},
		{
			name:       "contact.received",
			ev:         testCloudEvent(events.TypeContactReceived, map[string]any{"id": 21, "name": "Jane", "email": "j@b.c", "subject": "Hi"}),
			action:     "contact",
			entityType: "contact",
			entityID:   21,
			userName:   "Jane",
		},
		{
			name:       "settings.updated",
			ev:         testCloudEvent(events.TypeSettingsUpdated, map[string]any{"siteName": "Omoikane", "blogEnabled": true}),
			action:     "update",
			entityType: "settings",
			entityID:   1,
			userName:   "system",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry, mapped, err := mapEventToLog(tc.ev)
			if err != nil {
				t.Fatalf("mapEventToLog error: %v", err)
			}
			if !mapped {
				t.Fatal("expected mapped=true")
			}
			if entry.EventID != tc.ev.ID {
				t.Errorf("EventID = %q, want %q", entry.EventID, tc.ev.ID)
			}
			if entry.Action != tc.action {
				t.Errorf("Action = %q, want %q", entry.Action, tc.action)
			}
			if entry.EntityType != tc.entityType {
				t.Errorf("EntityType = %q, want %q", entry.EntityType, tc.entityType)
			}
			if entry.EntityID != tc.entityID {
				t.Errorf("EntityID = %d, want %d", entry.EntityID, tc.entityID)
			}
			if entry.UserName != tc.userName {
				t.Errorf("UserName = %q, want %q", entry.UserName, tc.userName)
			}
			if entry.Detail == "" {
				t.Error("expected non-empty Detail")
			}
		})
	}
}

func TestMapEventToLog_UnknownTypeIgnored(t *testing.T) {
	ev := testCloudEvent("org.omoikane.future.out.of.scope.v1", map[string]any{"id": 1})
	entry, mapped, err := mapEventToLog(ev)
	if err != nil {
		t.Fatalf("unknown type should not error, got %v", err)
	}
	if mapped {
		t.Error("expected mapped=false for unknown type")
	}
	if entry.Action != "" || entry.EntityType != "" {
		t.Errorf("expected zero-worthy entry for unknown type, got %+v", entry)
	}
}

func TestMapEventToLog_MalformedPayloadErrors(t *testing.T) {
	ev := events.CloudEvent{ID: "bad", Type: events.TypePostPublished, Data: []byte("{not json")}
	_, mapped, err := mapEventToLog(ev)
	if err == nil {
		t.Fatal("expected error for malformed payload on a mapped type")
	}
	if mapped {
		t.Error("expected mapped=false when payload is malformed")
	}
}

func TestAuditEventHandler_IdempotentRedelivery(t *testing.T) {
	db := setupAuditService(t)
	h := &auditEventHandler{db: db}

	ev := testCloudEvent(events.TypePostPublished, map[string]any{"id": 42, "slug": "dup", "title": "Dup"})
	for i := 0; i < 2; i++ {
		if err := h.Handle(context.Background(), ev); err != nil {
			t.Fatalf("Handle attempt %d: %v", i+1, err)
		}
	}

	var count int64
	if err := db.Model(&models.AuditLog{}).Where("event_id = ?", ev.ID).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after redelivery, got %d", count)
	}
}

func TestAuditEventHandler_UnknownTypeStoresNothing(t *testing.T) {
	db := setupAuditService(t)
	h := &auditEventHandler{db: db}

	ev := testCloudEvent("org.omoikane.unknown.thing.v1", map[string]any{"id": 1})
	if err := h.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle unknown type: %v", err)
	}
	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows for unknown type, got %d", count)
	}
}

// TestAuditEventHandler_DetailContent ensures the stored row carries the
// human-readable detail the admin UI displays (Phase 32 gate data path).
func TestAuditEventHandler_DetailContent(t *testing.T) {
	db := setupAuditService(t)
	h := &auditEventHandler{db: db}

	ev := testCloudEvent(events.TypePostPublished, map[string]any{"id": 77, "slug": "phase-32", "title": "Phase 32 Post"})
	if err := h.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	var row models.AuditLog
	if err := db.Where("event_id = ?", ev.ID).First(&row).Error; err != nil {
		t.Fatalf("fetch row: %v", err)
	}
	if !strings.Contains(row.Detail, "Phase 32 Post") || !strings.Contains(row.Detail, "phase-32") {
		t.Errorf("Detail %q missing title/slug", row.Detail)
	}
	if row.Action != "publish" || row.EntityType != "post" || row.EntityID != 77 {
		t.Errorf("unexpected row: %+v", row)
	}
}
