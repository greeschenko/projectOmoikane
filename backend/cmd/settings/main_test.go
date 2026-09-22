package main

import (
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

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func settingsTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// setupSettingsService boots the real settings-service mux (newSettingsMux)
// against the test database, mirroring production (outbox wired, noop cache).
func setupSettingsService(t *testing.T) (*gorm.DB, *httptest.Server) {
	t.Helper()
	db, err := database.Connect(settingsTestDSN())
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
		Outbox:    events.NewGormOutboxStore(db),
	}
	s := httptest.NewServer(newSettingsMux(h))
	t.Cleanup(s.Close)
	return db, s
}

// sessionCookie mints a JWT and returns it as the "session" cookie.
func sessionCookie(t *testing.T, userID uint, role string) map[string][]string {
	t.Helper()
	token, err := auth.GenerateToken(userID, role, "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	return map[string][]string{"Cookie": {"session=" + token}}
}

func TestSettingsService_Health(t *testing.T) {
	_, s := setupSettingsService(t)

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

// TestSettingsService_PublicGet: GET /settings must be public (gateway flips
// /api/settings here; the public header + favicon + blog/setup pages read it).
func TestSettingsService_PublicGet(t *testing.T) {
	db, s := setupSettingsService(t)
	db.Create(&models.SiteSetting{ID: 1, SiteName: "Omoikane Test", Tagline: "t"})

	resp, err := http.Get(s.URL + "/settings")
	if err != nil {
		t.Fatalf("GET /settings: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from GET /settings, got %d", resp.StatusCode)
	}
	body := readBody(resp)
	if !strings.Contains(body, "Omoikane Test") {
		t.Errorf("settings body missing site name: %s", body)
	}
}

// TestSettingsService_UpdateAdminOnly: PUT /settings requires an admin; 401
// anonymous, 403 non-admin, 200 admin.
func TestSettingsService_UpdateAdminOnly(t *testing.T) {
	db, s := setupSettingsService(t)
	user := models.User{Name: "U", Email: "u@t.com", Password: "x", Role: "user", Status: "active"}
	admin := models.User{Name: "A", Email: "a@t.com", Password: "x", Role: "admin", Status: "active"}
	db.Create(&user)
	db.Create(&admin)

	body := `{"siteName":"Renamed","tagline":"new","description":"d"}`
	put := func(cookie map[string][]string) int {
		t.Helper()
		req, _ := http.NewRequest("PUT", s.URL+"/settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range cookie {
			req.Header[k] = v
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("PUT /settings: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if code := put(nil); code != http.StatusUnauthorized {
		t.Errorf("anonymous PUT /settings: expected 401, got %d", code)
	}
	if code := put(sessionCookie(t, user.ID, "user")); code != http.StatusForbidden {
		t.Errorf("user PUT /settings: expected 403, got %d", code)
	}
	if code := put(sessionCookie(t, admin.ID, "admin")); code != http.StatusOK {
		t.Errorf("admin PUT /settings: expected 200, got %d", code)
	}
}
