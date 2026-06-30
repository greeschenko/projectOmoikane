package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func contactServerRecaptcha(db *gorm.DB, recaptchaSecret string) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret, RecaptchaSecret: recaptchaSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /contact", h.SubmitContact)
	mux.HandleFunc("GET /contacts", h.Admin(h.GetContacts))
	mux.HandleFunc("GET /contacts/{id}", h.Admin(h.GetContact))
	mux.HandleFunc("POST /contacts/{id}/read", h.Admin(h.MarkContactRead))
	mux.HandleFunc("DELETE /contacts/{id}", h.Admin(h.DeleteContact))
	return httptest.NewServer(mux)
}

func contactServer(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()
	return contactServerRecaptcha(db, "")
}

func TestSubmitContact_Success(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	body := `{"name":"Alice","email":"alice@test.com","subject":"Hello","message":"I have a question"}`
	resp, err := http.Post(s.URL+"/contact", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.ContactMessage{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 contact message, got %d", count)
	}
}

func TestSubmitContact_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	tests := []struct {
		name string
		body string
	}{
		{"empty", `{}`},
		{"no name", `{"email":"a@b.com","message":"hi"}`},
		{"no email", `{"name":"Alice","message":"hi"}`},
		{"no message", `{"name":"Alice","email":"a@b.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(s.URL+"/contact", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 400 {
				t.Errorf("Expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestSubmitContact_DefaultsSubject(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	body := `{"name":"Bob","email":"bob@test.com","message":"No subject"}`
	resp, err := http.Post(s.URL+"/contact", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	var msg models.ContactMessage
	db.First(&msg)
	if msg.Subject != "Contact Form Submission" {
		t.Errorf("Expected default subject, got %s", msg.Subject)
	}
}

func TestGetContacts_AdminOnly(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass123!", "admin")
	createTestUser(db, "User", "user@test.com", "Pass123!", "user")

	db.Create(&models.ContactMessage{
		Name: "Alice", Email: "alice@test.com",
		Subject: "Hello", Message: "Hi",
	})

	// User should get 403
	userCookie := loginAs(t, s, "user@test.com", "Pass123!")
	resp := authenticatedRequest(t, "GET", s.URL+"/contacts", "", userCookie)
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403 for non-admin, got %d", resp.StatusCode)
	}

	// Admin should get 200
	adminCookie := loginAs(t, s, "admin@test.com", "Pass123!")
	resp = authenticatedRequest(t, "GET", s.URL+"/contacts", "", adminCookie)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 for admin, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	contacts := data["contacts"].([]interface{})
	if len(contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(contacts))
	}
}

func TestMarkContactRead(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass123!", "admin")
	db.Create(&models.ContactMessage{
		Name: "Alice", Email: "alice@test.com",
		Subject: "Hello", Message: "Hi",
	})

	cookie := loginAs(t, s, "admin@test.com", "Pass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/contacts/1/read", "", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var msg models.ContactMessage
	db.First(&msg, 1)
	if !msg.Read {
		t.Error("Expected message to be marked as read")
	}
}

func TestDeleteContact(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass123!", "admin")
	db.Create(&models.ContactMessage{
		Name: "Alice", Email: "alice@test.com",
		Subject: "Hello", Message: "Hi",
	})

	cookie := loginAs(t, s, "admin@test.com", "Pass123!")

	resp := authenticatedRequest(t, "DELETE", s.URL+"/contacts/1", "", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.ContactMessage{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 contacts after delete, got %d", count)
	}
}

func TestGetContact_NotFound(t *testing.T) {
	db := setupTestDB(t)
	s := contactServer(t, db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass123!")

	resp := authenticatedRequest(t, "GET", s.URL+"/contacts/999", "", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}
