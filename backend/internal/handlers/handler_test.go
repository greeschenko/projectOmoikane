package handlers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"omoikane-backend/internal/database"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const testJWTSecret = "test-secret"

func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := getTestDSN()
	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test DB: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("Failed to migrate test DB: %v", err)
	}
	t.Cleanup(func() {
		tables := []string{
			"users", "pages", "blog_posts", "tags", "blog_post_tags",
			"categories", "likes", "media_items", "messages", "site_settings",
			"password_reset_tokens", "contact_messages",
		}
		for _, table := range tables {
			db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
		}
	})
	return db
}

func authServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /setup", h.Setup)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/logout", h.Logout)
	mux.HandleFunc("POST /auth/forgot-password", h.ForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", h.ResetPassword)
	return httptest.NewServer(mux)
}

func userServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /users", h.Admin(h.GetUsers))
	mux.HandleFunc("POST /users", h.Admin(h.CreateUser))
	mux.HandleFunc("PUT /users/{id}", h.Admin(h.UpdateUser))
	mux.HandleFunc("DELETE /users/{id}", h.Admin(h.DeleteUser))
	return httptest.NewServer(mux)
}

func createTestUser(db *gorm.DB, name, email, password, role string) models.User {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := models.User{
		Name:     name,
		Email:    email,
		Password: string(hashed),
		Role:     role,
		Status:   "active",
	}
	db.Create(&user)
	return user
}

func loginAs(t *testing.T, server *httptest.Server, email, password string) *http.Cookie {
	t.Helper()
	body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	req, err := http.NewRequest("POST", server.URL+"/auth/login", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login failed (status %d): %s", resp.StatusCode, string(b))
	}
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("No session cookie in login response")
	return nil
}

func authenticatedRequest(t *testing.T, method, url, body string, cookie *http.Cookie) *http.Response {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	resp.Body.Close()
	return string(b)
}

func decodeJSON(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("Failed to decode JSON: %v\nBody: %s", err, data)
	}
	return result
}

func decodeJSONArray(t *testing.T, data string) []interface{} {
	t.Helper()
	var result []interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("Failed to decode JSON array: %v\nBody: %s", err, data)
	}
	return result
}
