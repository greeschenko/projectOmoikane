package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func dashboardServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /dashboard", h.Admin(h.GetDashboard))
	return httptest.NewServer(mux)
}

func TestDashboard_ReturnsCounts(t *testing.T) {
	db := setupTestDB(t)
	s := dashboardServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	createTestUser(db, "User1", "user1@test.com", "Pass1234!", "user")
	createTestUser(db, "User2", "user2@test.com", "Pass1234!", "user")

	db.Create(&models.Page{Title: "Home", Slug: "home", Status: "published", PreviewToken: "tok1"})
	db.Create(&models.BlogPost{Title: "Post", Slug: "post", AuthorID: 1, Status: "published"})
	db.Create(&models.Message{Title: "Msg", Content: "Hello", ReadBy: "[]"})

	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	resp := authenticatedRequest(t, "GET", s.URL+"/dashboard", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["users"] != float64(3) {
		t.Errorf("Expected 3 users, got %v", data["users"])
	}
	if data["pages"] != float64(1) {
		t.Errorf("Expected 1 page, got %v", data["pages"])
	}
	if data["posts"] != float64(1) {
		t.Errorf("Expected 1 post, got %v", data["posts"])
	}
	if data["messages"] != float64(1) {
		t.Errorf("Expected 1 message, got %v", data["messages"])
	}
}

func TestDashboard_RequiresAdmin(t *testing.T) {
	db := setupTestDB(t)
	s := dashboardServer(db)
	defer s.Close()

	createTestUser(db, "User", "user@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "user@test.com", "Pass1234!")

	resp := authenticatedRequest(t, "GET", s.URL+"/dashboard", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}
