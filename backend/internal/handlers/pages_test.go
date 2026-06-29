package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func pagesServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /pages", h.GetPages)
	mux.HandleFunc("GET /pages/{id}", h.GetPage)
	mux.HandleFunc("GET /pages/slug/{slug}", h.GetPageBySlug)
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("PUT /pages/{id}", h.Auth(h.UpdatePage))
	mux.HandleFunc("DELETE /pages/{id}", h.Auth(h.DeletePage))
	mux.HandleFunc("PUT /pages/reorder", h.Auth(h.ReorderPages))
	return httptest.NewServer(mux)
}

func createTestPage(db *gorm.DB, title, slug, status string, sortOrder int) models.Page {
	page := models.Page{
		Title:           title,
		Slug:            slug,
		Content:         fmt.Sprintf("<p>%s content</p>", title),
		Status:          status,
		SortOrder:       sortOrder,
		InMenu:          true,
		MetaTitle:       title,
		MetaDescription: title + " description",
		PreviewToken:    fmt.Sprintf("preview-%s", slug),
	}
	db.Create(&page)
	return page
}

func TestGetPages_ListPublished(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestPage(db, "Home", "home", "published", 0)
	createTestPage(db, "About", "about", "published", 1)

	resp, err := http.Get(s.URL + "/pages")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	pages := decodeJSONArray(t, readBody(t, resp))
	if len(pages) != 2 {
		t.Errorf("Expected 2 pages, got %d", len(pages))
	}
}

func TestGetPageBySlug_ReturnsParentFields(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	parent := createTestPage(db, "Parent Page", "parent", "published", 0)
	child := models.Page{
		Title:    "Child Page",
		Slug:     "child",
		Content:  "<p>Child content</p>",
		Status:   "published",
		ParentID: &parent.ID,
	}
	db.Create(&child)

	resp, err := http.Get(s.URL + "/pages/slug/child")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}

	if result["parentTitle"] != "Parent Page" {
		t.Errorf("Expected parentTitle 'Parent Page', got %v", result["parentTitle"])
	}
	if result["parentSlug"] != "parent" {
		t.Errorf("Expected parentSlug 'parent', got %v", result["parentSlug"])
	}
}

func TestGetPages_ExcludesDrafts(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestPage(db, "Published", "pub", "published", 0)
	createTestPage(db, "Draft", "draft", "draft", 1)

	resp, err := http.Get(s.URL + "/pages")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	pages := decodeJSONArray(t, readBody(t, resp))
	if len(pages) != 1 {
		t.Errorf("Expected 1 published page, got %d", len(pages))
	}
}

func TestReorderPages_UpdatesOrder(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	p1 := createTestPage(db, "First", "first", "published", 0)
	p2 := createTestPage(db, "Second", "second", "published", 1)

	body := fmt.Sprintf(`{"pageIds":[%d,%d]}`, p2.ID, p1.ID)
	resp := authenticatedRequest(t, "PUT", s.URL+"/pages/reorder", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify order
	resp2, _ := http.Get(s.URL + "/pages")
	defer resp2.Body.Close()
	pages := decodeJSONArray(t, readBody(t, resp2))
	if len(pages) < 2 {
		t.Fatal("Expected at least 2 pages")
	}
	firstPage := pages[0].(map[string]interface{})
	if firstPage["id"] != float64(p2.ID) {
		t.Errorf("Expected first page ID %d, got %v", p2.ID, firstPage["id"])
	}
}
