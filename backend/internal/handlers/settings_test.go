package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"omoikane-backend/internal/handlers"

	"gorm.io/gorm"
)

func settingsServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /settings", h.GetSettings) // public
	mux.HandleFunc("PUT /settings", h.Admin(h.UpdateSettings))
	mux.HandleFunc("GET /settings/profile", h.Auth(h.GetProfile))
	mux.HandleFunc("PUT /settings/profile", h.Auth(h.UpdateProfile))
	mux.HandleFunc("POST /settings/password", h.Auth(h.ChangePassword))
	return httptest.NewServer(mux)
}

func TestGetSettings_Public(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	resp, err := http.Get(s.URL + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 (public), got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if _, ok := data["siteName"]; !ok {
		t.Error("Expected siteName in response")
	}
}

func TestGetSettings_ReturnsDefaults(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	resp, err := http.Get(s.URL + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["siteName"] != "Omoikane" {
		t.Errorf("Expected siteName Omoikane, got %v", data["siteName"])
	}
	if data["blogEnabled"] != true {
		t.Errorf("Expected blogEnabled true, got %v", data["blogEnabled"])
	}
}

func TestUpdateSettings_AdminOnly(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Regular", "user@test.com", "UserPass1!", "user")
	cookie := loginAs(t, s, "user@test.com", "UserPass1!")

	body := `{"siteName":"Hacked"}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/settings", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestGetSettings_ReturnsEmailTemplateFields(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	resp, err := http.Get(s.URL + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if _, ok := data["resetEmailSubject"]; !ok {
		t.Error("Expected resetEmailSubject in response")
	}
	if _, ok := data["resetEmailBodyHTML"]; !ok {
		t.Error("Expected resetEmailBodyHTML in response")
	}
	if data["resetEmailSubject"] != "Password Reset Request" {
		t.Errorf("Expected default resetEmailSubject 'Password Reset Request', got %v", data["resetEmailSubject"])
	}
}

func TestUpdateSettings_UpdatesEmailTemplateFields(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"resetEmailSubject":"Custom Subject","resetEmailBodyHTML":"<p>Custom body {{.ResetLink}}</p>"}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/settings", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["resetEmailSubject"] != "Custom Subject" {
		t.Errorf("Expected resetEmailSubject 'Custom Subject', got %v", data["resetEmailSubject"])
	}
	if data["resetEmailBodyHTML"] != "<p>Custom body {{.ResetLink}}</p>" {
		t.Errorf("Unexpected resetEmailBodyHTML: %v", data["resetEmailBodyHTML"])
	}
}

func TestUpdateSettings_PartialUpdate(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"siteName":"My Site","blogEnabled":false}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/settings", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify the update persisted
	resp2 := authenticatedRequest(t, "GET", s.URL+"/settings", "", cookie)
	defer resp2.Body.Close()
	data := decodeJSON(t, readBody(t, resp2))
	if data["siteName"] != "My Site" {
		t.Errorf("Expected siteName 'My Site', got %v", data["siteName"])
	}
	if data["blogEnabled"] != false {
		t.Errorf("Expected blogEnabled false, got %v", data["blogEnabled"])
	}
}

func TestGetProfile_ReturnsCurrentUser(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Test User", "test@test.com", "TestPass1!", "user")
	cookie := loginAs(t, s, "test@test.com", "TestPass1!")

	resp := authenticatedRequest(t, "GET", s.URL+"/settings/profile", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["name"] != "Test User" {
		t.Errorf("Expected name 'Test User', got %v", data["name"])
	}
	if data["email"] != "test@test.com" {
		t.Errorf("Expected email 'test@test.com', got %v", data["email"])
	}
}

func TestUpdateProfile_UpdatesNameAndEmail(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Old Name", "old@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "old@test.com", "Pass1234!")

	body := `{"name":"New Name","email":"new@test.com"}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/settings/profile", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Log in with new email
	cookie2 := loginAs(t, s, "new@test.com", "Pass1234!")
	resp2 := authenticatedRequest(t, "GET", s.URL+"/settings/profile", "", cookie2)
	defer resp2.Body.Close()
	data := decodeJSON(t, readBody(t, resp2))
	if data["name"] != "New Name" {
		t.Errorf("Expected name 'New Name', got %v", data["name"])
	}
}

func TestUpdateProfile_EmptyNameRejected(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Test", "test@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "test@test.com", "Pass1234!")

	body := `{"name":""}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/settings/profile", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_Success(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Test", "test@test.com", "OldPass1!", "user")
	cookie := loginAs(t, s, "test@test.com", "OldPass1!")

	body := `{"currentPassword":"OldPass1!","newPassword":"NewPass123!"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/settings/password", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Login with new password
	cookie2 := loginAs(t, s, "test@test.com", "NewPass123!")
	if cookie2 == nil {
		t.Error("Expected successful login with new password")
	}
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Test", "test@test.com", "OldPass1!", "user")
	cookie := loginAs(t, s, "test@test.com", "OldPass1!")

	body := `{"currentPassword":"WrongPass!","newPassword":"NewPass123!"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/settings/password", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestChangePassword_TooShort(t *testing.T) {
	db := setupTestDB(t)
	s := settingsServer(db)
	defer s.Close()

	createTestUser(db, "Test", "test@test.com", "OldPass1!", "user")
	cookie := loginAs(t, s, "test@test.com", "OldPass1!")

	body := `{"currentPassword":"OldPass1!","newPassword":"abc"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/settings/password", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
