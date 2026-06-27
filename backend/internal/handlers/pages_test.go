package handlers_test

import (
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

	data := decodeJSON(t, readBody(t, resp))
	pages, ok := data["pages"].([]interface{})
	if !ok {
		t.Fatal("Expected 'pages' array in response")
	}
	if len(pages) != 2 {
		t.Errorf("Expected 2 pages, got %d", len(pages))
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

	data := decodeJSON(t, readBody(t, resp))
	pages := data["pages"].([]interface{})
	if len(pages) != 1 {
		t.Errorf("Expected 1 published page, got %d", len(pages))
	}
}

func TestCreatePage_AuthRequired(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	resp, err := http.Post(s.URL+"/pages", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestCreatePage_CreatesPage(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author@test.com", "Pass1234!")

	body := `{"title":"My Page","slug":"my-page","content":"<p>Hello</p>"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/pages", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["title"] != "My Page" {
		t.Errorf("Expected title 'My Page', got %v", data["title"])
	}
	if data["status"] != "draft" {
		t.Errorf("Expected default status 'draft', got %v", data["status"])
	}
}

func TestCreatePage_GeneratesPreviewToken(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author@test.com", "Pass1234!")

	body := `{"title":"Preview Page","slug":"preview"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/pages", body, cookie)
	defer resp.Body.Close()

	data := decodeJSON(t, readBody(t, resp))
	if data["previewToken"] == "" {
		t.Error("Expected non-empty previewToken")
	}
}

func TestCreatePage_MissingTitle(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author2@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author2@test.com", "Pass1234!")

	body := `{"slug":"no-title"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/pages", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePage_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestPage(db, "Existing", "my-slug", "published", 0)

	createTestUser(db, "Author", "author3@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author3@test.com", "Pass1234!")

	body := `{"title":"Duplicate","slug":"my-slug"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/pages", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 409 {
		t.Errorf("Expected 409, got %d", resp.StatusCode)
	}
}

func TestGetPageBySlug_Resolves(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestPage(db, "About Us", "about-us", "published", 0)

	resp, err := http.Get(s.URL + "/pages/slug/about-us")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["title"] != "About Us" {
		t.Errorf("Expected title 'About Us', got %v", data["title"])
	}
}

func TestGetPageBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	resp, err := http.Get(s.URL + "/pages/slug/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdatePage_UpdatesFields(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author4@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author4@test.com", "Pass1234!")

	page := createTestPage(db, "Old Title", "old-slug", "draft", 0)

	updateBody := `{"title":"New Title","status":"published"}`
	resp := authenticatedRequest(t, "PUT", fmt.Sprintf("%s/pages/%d", s.URL, page.ID), updateBody, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["title"] != "New Title" {
		t.Errorf("Expected title 'New Title', got %v", data["title"])
	}
	if data["status"] != "published" {
		t.Errorf("Expected status 'published', got %v", data["status"])
	}
}

func TestUpdatePage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author5@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author5@test.com", "Pass1234!")

	body := `{"title":"Ghost"}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/pages/99999", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeletePage_SoftDeletes(t *testing.T) {
	db := setupTestDB(t)
	s := pagesServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author6@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author6@test.com", "Pass1234!")

	page := createTestPage(db, "Delete Me", "delete-me", "published", 0)

	resp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/pages/%d", s.URL, page.ID), "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify deleted page is not in public list
	resp2, _ := http.Get(s.URL + "/pages")
	defer resp2.Body.Close()
	data := decodeJSON(t, readBody(t, resp2))
	pages := data["pages"].([]interface{})
	if len(pages) != 0 {
		t.Errorf("Expected 0 pages after delete, got %d", len(pages))
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
	data := decodeJSON(t, readBody(t, resp2))
	pages := data["pages"].([]interface{})
	if len(pages) < 2 {
		t.Fatal("Expected at least 2 pages")
	}
	firstPage := pages[0].(map[string]interface{})
	if firstPage["id"] != float64(p2.ID) {
		t.Errorf("Expected first page ID %d, got %v", p2.ID, firstPage["id"])
	}
}
