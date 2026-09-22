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
	return pendingOutboxRowsOfType(t, db, "")
}

// pendingOutboxRowsOfType returns pending outbox rows of a single event type
// ("" = all types). Used because loginAs now emits auth.login, so tests
// asserting "exactly one content event" must ignore the login row.
func pendingOutboxRowsOfType(t *testing.T, db *gorm.DB, typ string) []events.OutboxEvent {
	t.Helper()
	q := db.Where("status = ?", events.OutboxPending).Order("id ASC")
	if typ != "" {
		q = q.Where("event_type = ?", typ)
	}
	var rows []events.OutboxEvent
	if err := q.Find(&rows).Error; err != nil {
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

	rows := pendingOutboxRowsOfType(t, db, events.TypeUserRegistered)
	if len(rows) != 1 {
		t.Fatalf("expected 1 user.registered outbox row, got %d", len(rows))
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

// TestLogin_EnqueuesAuthLoginOutboxEvent covers the Phase 32 auth.login
// emission: a successful Login appends an auth.login CloudEvent, replacing the
// retired HTTP audit.Emit call.
func TestLogin_EnqueuesAuthLoginOutboxEvent(t *testing.T) {
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
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")
	if cookie == nil {
		t.Fatal("expected session cookie from login")
	}

	rows := pendingOutboxRowsOfType(t, db, events.TypeAuthLogin)
	if len(rows) != 1 {
		t.Fatalf("expected 1 auth.login outbox row, got %d", len(rows))
	}
	if rows[0].Source != events.SourceAuth || rows[0].Subject != "user/1" {
		t.Errorf("unexpected row: source=%s subject=%s", rows[0].Source, rows[0].Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(rows[0].Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	var data struct {
		ID     uint   `json:"id"`
		Email  string `json:"email"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.ID != 1 || data.Email != "admin@test.com" || data.Method != "cookie" {
		t.Errorf("unexpected auth.login payload: %+v", data)
	}
}

// TestSubmitContact_EnqueuesContactReceivedOutboxEvent covers the Phase 32
// contact.received emission from the public contact form.
func TestSubmitContact_EnqueuesContactReceivedOutboxEvent(t *testing.T) {
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
	mux.HandleFunc("POST /contact", h.SubmitContact)
	s := httptest.NewServer(mux)
	defer s.Close()

	resp, err := http.Post(s.URL+"/contact", "application/json",
		strings.NewReader(`{"name":"Jane","email":"jane@test.com","subject":"Hello","message":"Hi there"}`))
	if err != nil {
		t.Fatalf("contact request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRowsOfType(t, db, events.TypeContactReceived)
	if len(rows) != 1 {
		t.Fatalf("expected 1 contact.received outbox row, got %d", len(rows))
	}
	if rows[0].Source != events.SourceMessages || rows[0].Subject != "contact/1" {
		t.Errorf("unexpected row: source=%s subject=%s", rows[0].Source, rows[0].Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(rows[0].Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	var data struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.ID != 1 || data.Name != "Jane" || data.Email != "jane@test.com" ||
		data.Subject != "Hello" || data.Message != "Hi there" {
		t.Errorf("unexpected contact.received payload: %+v", data)
	}
}

// TestUpdateSettings_EnqueuesSettingsUpdatedOutboxEvent covers the Phase 32
// settings.updated emission from the admin settings save.
func TestUpdateSettings_EnqueuesSettingsUpdatedOutboxEvent(t *testing.T) {
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
	mux.HandleFunc("PUT /settings", h.Admin(h.UpdateSettings))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "PUT", s.URL+"/settings",
		`{"siteName":"New Site"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRowsOfType(t, db, events.TypeSettingsUpdated)
	if len(rows) != 1 {
		t.Fatalf("expected 1 settings.updated outbox row, got %d", len(rows))
	}
	if rows[0].Source != events.SourceSettings || rows[0].Subject != "settings/1" {
		t.Errorf("unexpected row: source=%s subject=%s", rows[0].Source, rows[0].Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(rows[0].Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	var data struct {
		SiteName    string `json:"siteName"`
		BlogEnabled bool   `json:"blogEnabled"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.SiteName != "New Site" || !data.BlogEnabled {
		t.Errorf("unexpected settings.updated payload: %+v", data)
	}
}
