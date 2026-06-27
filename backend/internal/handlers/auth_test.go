package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"omoikane-backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func TestSetup_CreatesAdmin(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	body := `{"email":"admin@test.com","password":"SecurePass123!"}`
	resp, err := http.Post(s.URL+"/setup", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	result := decodeJSON(t, data)
	if result["success"] != true {
		t.Errorf("Expected success=true, got %v", result["success"])
	}

	var count int64
	db.Model(&models.User{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestSetup_RejectsWhenUsersExist(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "existing@test.com", "pass", "admin")

	body := `{"email":"admin2@test.com","password":"SecurePass123!"}`
	resp, err := http.Post(s.URL+"/setup", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSetup_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	resp, err := http.Post(s.URL+"/setup", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	createTestUser(db, "Test User", "user@test.com", "MyPass123!", "user")

	body := `{"email":"user@test.com","password":"MyPass123!"}`
	resp, err := http.Post(s.URL+"/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	hasSession := false
	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.Value != "" {
			hasSession = true
			break
		}
	}
	if !hasSession {
		t.Error("Expected session cookie")
	}

	data := readBody(t, resp)
	result := decodeJSON(t, data)
	if result["success"] != true {
		t.Errorf("Expected success=true, got %v", result["success"])
	}
	if result["role"] != "user" {
		t.Errorf("Expected role=user, got %v", result["role"])
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	createTestUser(db, "Test User", "user@test.com", "MyPass123!", "user")

	body := `{"email":"user@test.com","password":"WrongPassword!"}`
	resp, err := http.Post(s.URL+"/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestLogin_BannedUser(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	hashed, _ := bcrypt.GenerateFromPassword([]byte("MyPass123!"), bcrypt.DefaultCost)
	db.Exec("INSERT INTO users (created_at, updated_at, name, email, password, role, status) VALUES (NOW(), NOW(), ?, ?, ?, ?, ?)",
		"Banned User", "banned@test.com", string(hashed), "user", "banned")

	body := `{"email":"banned@test.com","password":"MyPass123!"}`
	resp, err := http.Post(s.URL+"/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestLogin_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	resp, err := http.Post(s.URL+"/auth/login", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_Success(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	body := `{"name":"New User","email":"new@test.com","password":"MyPass123!"}`
	resp, err := http.Post(s.URL+"/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.User{}).Where("email = ?", "new@test.com").Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	createTestUser(db, "Existing", "dup@test.com", "MyPass123!", "user")

	body := `{"name":"Duplicate","email":"dup@test.com","password":"MyPass123!"}`
	resp, err := http.Post(s.URL+"/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	resp, err := http.Post(s.URL+"/auth/register", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestLogout_ClearsSession(t *testing.T) {
	db := setupTestDB(t)
	s := authServer(db)
	defer s.Close()

	createTestUser(db, "Test User", "user@test.com", "MyPass123!", "user")
	cookie := loginAs(t, s, "user@test.com", "MyPass123!")

	req, err := http.NewRequest("POST", s.URL+"/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	foundLogout := false
	for _, c := range resp.Cookies() {
		if c.Name == "session" && c.MaxAge <= 0 {
			foundLogout = true
			break
		}
	}
	if !foundLogout {
		t.Error("Expected session cookie to be cleared")
	}
}
