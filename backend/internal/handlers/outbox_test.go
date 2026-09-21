package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"

	"gorm.io/gorm"
)

// registerRequest posts a valid register body and returns the response.
func registerRequest(t *testing.T, s *httptest.Server, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(s.URL+"/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	return resp
}

// pendingOutboxRows returns every outbox row still pending.
func pendingOutboxRows(t *testing.T, db *gorm.DB) []events.OutboxEvent {
	t.Helper()
	var rows []events.OutboxEvent
	if err := db.Where("status = ?", events.OutboxPending).Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("query outbox rows: %v", err)
	}
	return rows
}

func TestRegister_EnqueuesUserRegisteredOutboxEvent(t *testing.T) {
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	h := &handlers.Handler{
		DB:        db,
		JWTSecret: testJWTSecret,
		Outbox:    events.NewGormOutboxStore(db),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", h.Register)
	s := httptest.NewServer(mux)
	defer s.Close()

	resp := registerRequest(t, s, `{"name":"Jane","email":"jane@test.com","password":"SecurePass123!"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 pending outbox row, got %d", len(rows))
	}
	row := rows[0]
	if row.EventType != events.TypeUserRegistered {
		t.Errorf("EventType = %q, want %q", row.EventType, events.TypeUserRegistered)
	}
	if row.Source != events.SourceAuth {
		t.Errorf("Source = %q, want %q", row.Source, events.SourceAuth)
	}

	// Decode the stored CloudEvent envelope and validate its payload contract
	// (schemas/user.registered.json: {id, email, role}).
	ev, err := events.UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	if ev.ID == "" || ev.SpecVersion != "1.0" {
		t.Errorf("envelope id/specversion invalid: %+v", ev)
	}
	if ev.Subject != "user/1" {
		t.Errorf("Subject = %q, want user/1", ev.Subject)
	}
	var data struct {
		ID    uint   `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.ID == 0 || data.Email != "jane@test.com" || data.Role != "user" {
		t.Errorf("unexpected payload: %+v", data)
	}
}

func TestRegister_NoOutboxEventWhenNotWired(t *testing.T) {
	// The monolith leaves Handler.Outbox nil (single-writer): its Register must
	// NOT emit — no outbox row and the user still created.
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	s := authServer(db)
	defer s.Close()

	resp := registerRequest(t, s, `{"name":"Bob","email":"bob@test.com","password":"SecurePass123!"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 0 {
		t.Fatalf("expected 0 pending outbox rows (no outbox wiring), got %d", len(rows))
	}
}

func TestSetup_EnqueuesUserRegisteredOutboxEvent(t *testing.T) {
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	h := &handlers.Handler{
		DB:        db,
		JWTSecret: testJWTSecret,
		Outbox:    events.NewGormOutboxStore(db),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /setup", h.Setup)
	s := httptest.NewServer(mux)
	defer s.Close()

	resp, err := http.Post(s.URL+"/setup", "application/json", strings.NewReader(`{"email":"admin@test.com","password":"SecurePass123!"}`))
	if err != nil {
		t.Fatalf("setup request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(rows))
	}
	if rows[0].EventType != events.TypeUserRegistered || rows[0].Source != events.SourceAuth {
		t.Errorf("unexpected outbox row: type=%s source=%s", rows[0].EventType, rows[0].Source)
	}
}

func TestCreateUser_EnqueuesUserRegisteredOutboxEvent(t *testing.T) {
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	h := &handlers.Handler{
		DB:        db,
		JWTSecret: testJWTSecret,
		Outbox:    events.NewGormOutboxStore(db),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /users", h.Admin(h.CreateUser))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/users",
		`{"name":"Carol","email":"carol@test.com","password":"SecurePass123!","role":"user"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected create user success, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(rows))
	}
	if rows[0].Subject != "user/2" {
		t.Errorf("Subject = %q, want user/2 (admin is user 1)", rows[0].Subject)
	}
	var data struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	ev, err := events.UnmarshalCloudEvent([]byte(rows[0].Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.Email != "carol@test.com" || data.Role != "user" {
		t.Errorf("unexpected payload: %+v", data)
	}
}