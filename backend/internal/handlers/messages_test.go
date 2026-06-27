package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"omoikane-backend/internal/handlers"

	"gorm.io/gorm"
)

func messageServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /messages", h.Auth(h.GetMessages))
	mux.HandleFunc("POST /messages", h.Admin(h.CreateMessage))
	mux.HandleFunc("GET /messages/{id}", h.Auth(h.GetMessage))
	mux.HandleFunc("PUT /messages/{id}/read", h.Auth(h.MarkRead))
	mux.HandleFunc("DELETE /messages/{id}", h.Admin(h.DeleteMessage))
	return httptest.NewServer(mux)
}

func TestCreateMessage_AdminCreates(t *testing.T) {
	db := setupTestDB(t)
	s := messageServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	body := `{"title":"System Update","content":"Scheduled maintenance tonight."}`
	resp := authenticatedRequest(t, "POST", s.URL+"/messages", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["title"] != "System Update" {
		t.Errorf("Expected title 'System Update', got %v", data["title"])
	}
}

func TestCreateMessage_NonAdminRejected(t *testing.T) {
	db := setupTestDB(t)
	s := messageServer(db)
	defer s.Close()

	createTestUser(db, "User", "user@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "user@test.com", "Pass1234!")

	body := `{"title":"Spam","content":"Spam content"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/messages", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestGetMessages_ListsForUser(t *testing.T) {
	db := setupTestDB(t)
	s := messageServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	// Create a message
	authenticatedRequest(t, "POST", s.URL+"/messages", `{"title":"Notice"}`, cookie)

	// Login as regular user and list
	createTestUser(db, "User", "user@test.com", "Pass1234!", "user")
	cookie2 := loginAs(t, s, "user@test.com", "Pass1234!")

	resp := authenticatedRequest(t, "GET", s.URL+"/messages", "", cookie2)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	msgs, ok := data["messages"].([]interface{})
	if !ok {
		t.Fatal("Expected 'messages' array")
	}
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(msgs))
	}
}

func TestMarkRead_MarksAsRead(t *testing.T) {
	db := setupTestDB(t)
	s := messageServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	resp := authenticatedRequest(t, "POST", s.URL+"/messages", `{"title":"Read This"}`, cookie)
	data := decodeJSON(t, readBody(t, resp))
	msgID := int(data["id"].(float64))
	resp.Body.Close()

	// Login as user and mark as read
	createTestUser(db, "User", "user@test.com", "Pass1234!", "user")
	cookie2 := loginAs(t, s, "user@test.com", "Pass1234!")

	markResp := authenticatedRequest(t, "PUT", fmt.Sprintf("%s/messages/%d/read", s.URL, msgID), "", cookie2)
	defer markResp.Body.Close()

	if markResp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", markResp.StatusCode)
	}

	markData := decodeJSON(t, readBody(t, markResp))
	if markData["read"] != true {
		t.Errorf("Expected read=true, got %v", markData["read"])
	}
}

func TestDeleteMessage_AdminDeletes(t *testing.T) {
	db := setupTestDB(t)
	s := messageServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	resp := authenticatedRequest(t, "POST", s.URL+"/messages", `{"title":"Delete Me"}`, cookie)
	data := decodeJSON(t, readBody(t, resp))
	msgID := int(data["id"].(float64))
	resp.Body.Close()

	delResp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/messages/%d", s.URL, msgID), "", cookie)
	defer delResp.Body.Close()

	if delResp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", delResp.StatusCode)
	}

	// Verify it's gone
	getResp := authenticatedRequest(t, "GET", fmt.Sprintf("%s/messages/%d", s.URL, msgID), "", cookie)
	defer getResp.Body.Close()
	if getResp.StatusCode != 404 {
		t.Errorf("Expected 404 after delete, got %d", getResp.StatusCode)
	}
}
