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

func blogServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /blog/posts", h.GetPosts)
	mux.HandleFunc("GET /admin/blog/posts", h.Admin(h.GetAdminPosts))
	mux.HandleFunc("GET /blog/posts/{id}", h.GetPost)
	mux.HandleFunc("GET /blog/posts/slug/{slug}", h.GetPostBySlug)
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	mux.HandleFunc("PUT /blog/posts/{id}", h.Auth(h.UpdatePost))
	mux.HandleFunc("DELETE /blog/posts/{id}", h.Auth(h.DeletePost))
	mux.HandleFunc("POST /blog/posts/{id}/like", h.Auth(h.ToggleLike))
	mux.HandleFunc("GET /blog/tags", h.GetTags)
	mux.HandleFunc("POST /blog/tags", h.Admin(h.CreateTag))
	mux.HandleFunc("GET /blog/categories", h.GetCategories)
	mux.HandleFunc("POST /blog/categories", h.Admin(h.CreateCategory))
	mux.HandleFunc("DELETE /blog/categories/{id}", h.Admin(h.DeleteCategory))
	return httptest.NewServer(mux)
}

func createTestPost(db *gorm.DB, title, slug, status string, authorID uint) models.BlogPost {
	post := models.BlogPost{
		Title:    title,
		Slug:     slug,
		Content:  fmt.Sprintf("<p>%s content</p>", title),
		Excerpt:  title + " excerpt",
		AuthorID: authorID,
		Status:   status,
	}
	db.Create(&post)
	return post
}

func TestGetPosts_ListPublished(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author@test.com", "Pass1234!", "user")
	author := models.User{}
	db.Where("email = ?", "author@test.com").First(&author)

	createTestPost(db, "Post One", "post-one", "published", author.ID)
	createTestPost(db, "Post Two", "post-two", "published", author.ID)

	resp, err := http.Get(s.URL + "/blog/posts")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := readBody(t, resp)
	posts := decodeJSONArray(t, data)
	if len(posts) != 2 {
		t.Errorf("Expected 2 posts, got %d", len(posts))
	}
}

func TestGetPosts_ExcludesDrafts(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author2@test.com", "Pass1234!", "user")
	author := models.User{}
	db.Where("email = ?", "author2@test.com").First(&author)

	createTestPost(db, "Published", "pub", "published", author.ID)
	createTestPost(db, "Draft", "draft", "draft", author.ID)

	resp, _ := http.Get(s.URL + "/blog/posts")
	defer resp.Body.Close()
	data := readBody(t, resp)
	posts := decodeJSONArray(t, data)
	if len(posts) != 1 {
		t.Errorf("Expected 1 published post, got %d", len(posts))
	}
}

func TestGetAdminPosts_ReturnsAllStatuses(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin-blog@test.com", "Pass1234!", "admin")
	adminCookie := loginAs(t, s, "admin-blog@test.com", "Pass1234!")
	author := models.User{}
	db.Where("email = ?", "admin-blog@test.com").First(&author)

	createTestPost(db, "Published Post", "pub-post", "published", author.ID)
	createTestPost(db, "Draft Post", "draft-post", "draft", author.ID)

	resp := authenticatedRequest(t, "GET", s.URL+"/admin/blog/posts", "", adminCookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	posts := decodeJSONArray(t, readBody(t, resp))
	if len(posts) != 2 {
		t.Errorf("Expected 2 posts (published + draft), got %d", len(posts))
	}
}

func TestCreatePost_AuthRequired(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	resp, err := http.Post(s.URL+"/blog/posts", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestCreatePost_CreatesPost(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author3@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author3@test.com", "Pass1234!")

	body := `{"title":"My Post","slug":"my-post","content":"<p>Hello</p>","status":"draft"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/blog/posts", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["title"] != "My Post" {
		t.Errorf("Expected title 'My Post', got %v", data["title"])
	}
	if data["slug"] != "my-post" {
		t.Errorf("Expected slug 'my-post', got %v", data["slug"])
	}
}

func TestCreatePost_MissingTitle(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author4@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author4@test.com", "Pass1234!")

	body := `{"slug":"no-title"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/blog/posts", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestGetPostBySlug_Resolves(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author5@test.com", "Pass1234!", "user")
	author := models.User{}
	db.Where("email = ?", "author5@test.com").First(&author)
	createTestPost(db, "About Us", "about-us", "published", author.ID)

	resp, err := http.Get(s.URL + "/blog/posts/slug/about-us")
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

func TestGetPostBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	resp, err := http.Get(s.URL + "/blog/posts/slug/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestToggleLike_Toggles(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author6@test.com", "Pass1234!", "user")
	author := models.User{}
	db.Where("email = ?", "author6@test.com").First(&author)
	post := createTestPost(db, "Likeable", "likeable", "published", author.ID)

	createTestUser(db, "Liker", "liker@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "liker@test.com", "Pass1234!")

	// First like
	resp1 := authenticatedRequest(t, "POST", fmt.Sprintf("%s/blog/posts/%d/like", s.URL, post.ID), "", cookie)
	defer resp1.Body.Close()
	if resp1.StatusCode != 200 {
		t.Errorf("First like: expected 200, got %d", resp1.StatusCode)
	}
	data1 := decodeJSON(t, readBody(t, resp1))
	if data1["liked"] != true {
		t.Errorf("Expected liked=true after first like, got %v", data1["liked"])
	}
	if data1["count"] != float64(1) {
		t.Errorf("Expected count=1, got %v", data1["count"])
	}

	// Toggle off
	resp2 := authenticatedRequest(t, "POST", fmt.Sprintf("%s/blog/posts/%d/like", s.URL, post.ID), "", cookie)
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("Second like: expected 200, got %d", resp2.StatusCode)
	}
	data2 := decodeJSON(t, readBody(t, resp2))
	if data2["liked"] != false {
		t.Errorf("Expected liked=false after toggle off, got %v", data2["liked"])
	}
	if data2["count"] != float64(0) {
		t.Errorf("Expected count=0, got %v", data2["count"])
	}
}

func TestToggleLike_AuthRequired(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	resp, err := http.Post(s.URL+"/blog/posts/1/like", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestUpdatePost_UpdatesFields(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author7@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author7@test.com", "Pass1234!")

	author := models.User{}
	db.Where("email = ?", "author7@test.com").First(&author)
	post := createTestPost(db, "Old", "old-slug", "draft", author.ID)

	body := `{"title":"New Title","status":"published"}`
	resp := authenticatedRequest(t, "PUT", fmt.Sprintf("%s/blog/posts/%d", s.URL, post.ID), body, cookie)
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

func TestDeletePost_SoftDeletes(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Author", "author8@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "author8@test.com", "Pass1234!")

	author := models.User{}
	db.Where("email = ?", "author8@test.com").First(&author)
	post := createTestPost(db, "Delete Me", "delete-me", "published", author.ID)

	resp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/blog/posts/%d", s.URL, post.ID), "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	resp2, _ := http.Get(s.URL + "/blog/posts")
	defer resp2.Body.Close()
	data := readBody(t, resp2)
	posts := decodeJSONArray(t, data)
	if len(posts) != 0 {
		t.Errorf("Expected 0 posts after delete, got %d", len(posts))
	}
}

func TestCreateTag_AdminCreates(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	body := `{"name":"Technology","slug":"tech"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/blog/tags", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}
	data := decodeJSON(t, readBody(t, resp))
	if data["name"] != "Technology" {
		t.Errorf("Expected name 'Technology', got %v", data["name"])
	}
}

func TestCreateTag_NonAdminRejected(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "User", "user@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "user@test.com", "Pass1234!")

	body := `{"name":"Tech"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/blog/tags", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestGetTags_ListAll(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	db.Create(&models.Tag{Name: "Go", Slug: "go"})
	db.Create(&models.Tag{Name: "Rust", Slug: "rust"})

	resp, err := http.Get(s.URL + "/blog/tags")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	tags := decodeJSONArray(t, readBody(t, resp))
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestCreateCategory_AdminCreates(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	body := `{"name":"News","slug":"news"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/blog/categories", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}
	data := decodeJSON(t, readBody(t, resp))
	if data["name"] != "News" {
		t.Errorf("Expected name 'News', got %v", data["name"])
	}
}

func TestGetCategories_ListAll(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	db.Create(&models.Category{Name: "News", Slug: "news"})
	db.Create(&models.Category{Name: "Tutorials", Slug: "tutorials"})

	resp, err := http.Get(s.URL + "/blog/categories")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	cats := decodeJSONArray(t, readBody(t, resp))
	if len(cats) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(cats))
	}
}

func TestDeleteCategory_AdminDeletes(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin-cat@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin-cat@test.com", "Pass1234!")

	cat := models.Category{Name: "ToDelete", Slug: "to-delete"}
	db.Create(&cat)

	body := fmt.Sprintf(`{"id":%d}`, cat.ID)
	resp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/blog/categories/%d", s.URL, cat.ID), body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	data := decodeJSON(t, readBody(t, resp))
	if data["success"] != true {
		t.Errorf("Expected success=true, got %v", data["success"])
	}

	// Verify deleted
	resp2, _ := http.Get(s.URL + "/blog/categories")
	defer resp2.Body.Close()
	cats := decodeJSONArray(t, readBody(t, resp2))
	if len(cats) != 0 {
		t.Errorf("Expected 0 categories after delete, got %d", len(cats))
	}
}

func TestDeleteCategory_NonAdminRejected(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "User", "user-cat@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "user-cat@test.com", "Pass1234!")

	cat := models.Category{Name: "NoDelete", Slug: "no-delete"}
	db.Create(&cat)

	resp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/blog/categories/%d", s.URL, cat.ID), "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestUpdatePost_UpdatesTags(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin-tags@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin-tags@test.com", "Pass1234!")

	author := models.User{}
	db.Where("email = ?", "admin-tags@test.com").First(&author)

	// Create tags
	tag1 := models.Tag{Name: "Go", Slug: "go"}
	tag2 := models.Tag{Name: "Rust", Slug: "rust"}
	tag3 := models.Tag{Name: "Python", Slug: "python"}
	db.Create(&tag1)
	db.Create(&tag2)
	db.Create(&tag3)

	// Create post with tag1
	post := createTestPost(db, "Tagged Post", "tagged-post", "published", author.ID)
	db.Model(&post).Association("Tags").Append(&tag1)

	// Update to have tag2 and tag3
	body := fmt.Sprintf(`{"title":"Updated Tags","tags":["Rust","Python"]}`)
	resp := authenticatedRequest(t, "PUT", fmt.Sprintf("%s/blog/posts/%d", s.URL, post.ID), body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	tags := data["tags"].([]interface{})
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestUpdatePost_UpdatesCategory(t *testing.T) {
	db := setupTestDB(t)
	s := blogServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin-cat2@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin-cat2@test.com", "Pass1234!")

	author := models.User{}
	db.Where("email = ?", "admin-cat2@test.com").First(&author)

	cat1 := models.Category{Name: "News", Slug: "news"}
	cat2 := models.Category{Name: "Tutorials", Slug: "tutorials"}
	db.Create(&cat1)
	db.Create(&cat2)

	post := createTestPost(db, "Categorized", "categorized", "published", author.ID)
	db.Model(&post).Update("category_id", cat1.ID)

	body := fmt.Sprintf(`{"categoryId":%d}`, cat2.ID)
	resp := authenticatedRequest(t, "PUT", fmt.Sprintf("%s/blog/posts/%d", s.URL, post.ID), body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	if data["categoryId"] != float64(cat2.ID) {
		t.Errorf("Expected categoryId %d, got %v", cat2.ID, data["categoryId"])
	}
}
