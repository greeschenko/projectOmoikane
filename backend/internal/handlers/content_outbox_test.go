package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"
)

// TestCreatePage_Published_EnqueuesPagePublishedOutboxEvent covers the
// create-as-published path: a page.published CloudEvent must be enqueued inside
// the CreatePage transaction when the outbox is wired.
func TestCreatePage_Published_EnqueuesPagePublishedOutboxEvent(t *testing.T) {
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
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/pages",
		`{"title":"About","slug":"about","content":"<p>hi</p>","status":"published"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected page create success, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(rows))
	}
	row := rows[0]
	if row.EventType != events.TypePagePublished || row.Source != events.SourceContent {
		t.Errorf("unexpected row: type=%s source=%s", row.EventType, row.Source)
	}
	if row.Subject != "page/1" {
		t.Errorf("Subject = %q, want page/1", row.Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	if ev.Type != events.TypePagePublished {
		t.Errorf("envelope type = %q, want %q", ev.Type, events.TypePagePublished)
	}
	var data struct {
		ID          uint   `json:"id"`
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Status      string `json:"status"`
		PublishedAt string `json:"publishedAt"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.ID != 1 || data.Slug != "about" || data.Title != "About" || data.Status != "published" || data.PublishedAt == "" {
		t.Errorf("unexpected page.published payload: %+v", data)
	}
}

// TestCreatePage_Draft_EnqueuesNoOutboxEvent: a draft create never publishes.
func TestCreatePage_Draft_EnqueuesNoOutboxEvent(t *testing.T) {
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
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/pages",
		`{"title":"Draft","slug":"draft","content":"<p>wip</p>","status":"draft"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected page create success, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 0 {
		t.Fatalf("expected 0 pending rows for draft create, got %d", len(rows))
	}
}

// TestUpdatePage_PublishedTransition_Enqueues covers draft -> published.
func TestUpdatePage_PublishedTransition_Enqueues(t *testing.T) {
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
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("PUT /pages/{id}", h.Auth(h.UpdatePage))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	// Create as draft (no event).
	resp := authenticatedRequest(t, "POST", s.URL+"/pages",
		`{"title":"Draft","slug":"draft","content":"<p>wip</p>","status":"draft"}`, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("page create failed: %d", resp.StatusCode)
	}
	if rows := pendingOutboxRows(t, db); len(rows) != 0 {
		t.Fatalf("expected 0 rows after draft create, got %d", len(rows))
	}

	// Publish the page -> exactly one page.published row.
	resp = authenticatedRequest(t, "PUT", s.URL+"/pages/1",
		`{"status":"published"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected update success, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}
	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending row after publish transition, got %d", len(rows))
	}
	if rows[0].EventType != events.TypePagePublished || rows[0].Subject != "page/1" {
		t.Errorf("unexpected row: type=%s subject=%s", rows[0].EventType, rows[0].Subject)
	}
}

// TestUpdatePage_NoEmissionWhenAlreadyPublished: publishing an already
// published page is an update, not a publish — no extra event.
func TestUpdatePage_NoEmissionWhenAlreadyPublished(t *testing.T) {
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
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("PUT /pages/{id}", h.Auth(h.UpdatePage))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/pages",
		`{"title":"Live","slug":"live","content":"<p>x</p>","status":"published"}`, cookie)
	resp.Body.Close()
	if rows := pendingOutboxRows(t, db); len(rows) != 1 {
		t.Fatalf("expected 1 row from initial publish, got %d", len(rows))
	}

	resp = authenticatedRequest(t, "PUT", s.URL+"/pages/1",
		`{"title":"Live v2","status":"published"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update failed: %d", resp.StatusCode)
	}
	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected no additional row for already-published update, got %d", len(rows))
	}
}

// TestBatchPages_Publish_EnqueuesPerTransitionedPage covers the batch publish
// action: events only for pages that actually flip to published.
func TestBatchPages_Publish_EnqueuesPerTransitionedPage(t *testing.T) {
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
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	mux.HandleFunc("POST /pages/batch", h.Auth(h.BatchPages))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	// One published, one draft.
	for i, slug := range []string{"published", "draft"} {
		status := "draft"
		if i == 0 {
			status = "published"
		}
		resp := authenticatedRequest(t, "POST", s.URL+"/pages",
			fmt.Sprintf(`{"title":"%s","slug":"%s","content":"<p>x</p>","status":"%s"}`, slug, slug, status), cookie)
		resp.Body.Close()
	}
	// Clear baseline, then batch-publish both.
	if err := db.Where("status = ?", events.OutboxPending).Delete(&events.OutboxEvent{}).Error; err != nil {
		t.Fatalf("clear outbox: %v", err)
	}

	resp := authenticatedRequest(t, "POST", s.URL+"/pages/batch",
		`{"action":"publish","ids":[1,2]}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("batch publish failed: %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	// Page 1 was already published -> no event. Page 2 transitions -> 1 event.
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending row (only the draft transitioned), got %d", len(rows))
	}
	if rows[0].Subject != "page/2" {
		t.Errorf("Subject = %q, want page/2", rows[0].Subject)
	}
}

// TestCreatePost_Published_EnqueuesPostPublishedOutboxEvent covers the
// create-as-published post path with tags + category → tagIds payload.
func TestCreatePost_Published_EnqueuesPostPublishedOutboxEvent(t *testing.T) {
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
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	// Tags must already exist for the join (CreatePost does not create tags).
	db.Create(&models.Tag{Name: "go", Slug: "go"})
	db.Create(&models.Tag{Name: "events", Slug: "events"})

	resp := authenticatedRequest(t, "POST", s.URL+"/blog/posts",
		`{"title":"Post","slug":"post-1","content":"<p>x</p>","status":"published","tags":["go","events"]}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected post create success, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(rows))
	}
	row := rows[0]
	if row.EventType != events.TypePostPublished || row.Source != events.SourceContent {
		t.Errorf("unexpected row: type=%s source=%s", row.EventType, row.Source)
	}
	if row.Subject != "post/1" {
		t.Errorf("Subject = %q, want post/1", row.Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	var data struct {
		ID          uint   `json:"id"`
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Status      string `json:"status"`
		CategoryID  *uint  `json:"categoryId"`
		TagIDs      []uint `json:"tagIds"`
		PublishedAt string `json:"publishedAt"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.ID != 1 || data.Slug != "post-1" || data.Status != "published" {
		t.Errorf("unexpected post.published payload: %+v", data)
	}
	// Both known tags must appear in tagIds (1 = "go", 2 = "events").
	if len(data.TagIDs) != 2 || data.TagIDs[0] != 1 || data.TagIDs[1] != 2 {
		t.Errorf("expected tagIds [1 2], got %v", data.TagIDs)
	}
}

// TestUpdatePost_PublishedTransition_Enqueues covers draft -> published post.
func TestUpdatePost_PublishedTransition_Enqueues(t *testing.T) {
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
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	mux.HandleFunc("PUT /blog/posts/{id}", h.Auth(h.UpdatePost))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/blog/posts",
		`{"title":"Draft","slug":"draft-1","content":"<p>wip</p>","status":"draft"}`, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("post create failed: %d", resp.StatusCode)
	}
	if rows := pendingOutboxRows(t, db); len(rows) != 0 {
		t.Fatalf("expected 0 rows after draft post create, got %d", len(rows))
	}

	resp = authenticatedRequest(t, "PUT", s.URL+"/blog/posts/1",
		`{"status":"published"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected update success, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}
	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending row after post publish transition, got %d", len(rows))
	}
	if rows[0].EventType != events.TypePostPublished || rows[0].Subject != "post/1" {
		t.Errorf("unexpected row: type=%s subject=%s", rows[0].EventType, rows[0].Subject)
	}
}

// TestBatchPosts_Publish_EnqueuesPerTransitionedPost: batch publish emits for
// posts that flip to published.
func TestBatchPosts_Publish_EnqueuesPerTransitionedPost(t *testing.T) {
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
	mux.HandleFunc("POST /blog/posts", h.Auth(h.CreatePost))
	mux.HandleFunc("POST /blog/posts/batch", h.Auth(h.BatchPosts))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	statuses := []string{"published", "draft"}
	for i, st := range statuses {
		resp := authenticatedRequest(t, "POST", s.URL+"/blog/posts",
			fmt.Sprintf(`{"title":"Post %d","slug":"post-%d","content":"<p>x</p>","status":"%s"}`, i, i, st), cookie)
		resp.Body.Close()
	}
	if err := db.Where("status = ?", events.OutboxPending).Delete(&events.OutboxEvent{}).Error; err != nil {
		t.Fatalf("clear outbox: %v", err)
	}

	resp := authenticatedRequest(t, "POST", s.URL+"/blog/posts/batch",
		`{"action":"publish","ids":[1,2]}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("batch post publish failed: %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending row (only the draft transitioned), got %d", len(rows))
	}
	if rows[0].Subject != "post/2" {
		t.Errorf("Subject = %q, want post/2", rows[0].Subject)
	}
}

// TestUploadMedia_EnqueuesMediaUploadedOutboxEvent covers the media.uploaded
// emission from UploadMedia when the outbox is wired.
func TestUploadMedia_EnqueuesMediaUploadedOutboxEvent(t *testing.T) {
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	dir := t.TempDir()
	h := &handlers.Handler{
		DB:        db,
		JWTSecret: testJWTSecret,
		UploadDir: dir,
		Outbox:    events.NewGormOutboxStore(db),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /media", h.Auth(h.UploadMedia))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "pic.png")
	fw.Write(makeRealPNG())
	w.Close()

	req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", resp.StatusCode, readBody(t, resp))
	}

	rows := pendingOutboxRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending outbox row, got %d", len(rows))
	}
	row := rows[0]
	if row.EventType != events.TypeMediaUploaded || row.Source != events.SourceMedia {
		t.Errorf("unexpected row: type=%s source=%s", row.EventType, row.Source)
	}
	if row.Subject != "media/1" {
		t.Errorf("Subject = %q, want media/1", row.Subject)
	}
	ev, err := events.UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		t.Fatalf("unmarshal stored envelope: %v", err)
	}
	var data struct {
		ID         uint   `json:"id"`
		Filename   string `json:"filename"`
		Alt        string `json:"alt"`
		URL        string `json:"url"`
		ThumbURL   string `json:"thumbUrl"`
		Size       int64  `json:"size"`
		UploadedAt string `json:"uploadedAt"`
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}
	if data.ID != 1 || data.Filename != "pic.png" || data.URL == "" || data.Size == 0 || data.UploadedAt == "" {
		t.Errorf("unexpected media.uploaded payload: %+v", data)
	}
}

// TestCreatePage_NoOutboxEventWhenNotWired guards single-writer: the monolith
// (Outbox nil) must not emit page.published.
func TestCreatePage_NoOutboxEventWhenNotWired(t *testing.T) {
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret} // Outbox nil
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /pages", h.Auth(h.CreatePage))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	resp := authenticatedRequest(t, "POST", s.URL+"/pages",
		`{"title":"About","slug":"about","content":"<p>hi</p>","status":"published"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("page create failed: %d", resp.StatusCode)
	}
	if rows := pendingOutboxRows(t, db); len(rows) != 0 {
		t.Fatalf("expected 0 pending rows (no outbox wiring), got %d", len(rows))
	}
}

// TestUploadMedia_NoOutboxEventWhenNotWired guards single-writer for media.
func TestUploadMedia_NoOutboxEventWhenNotWired(t *testing.T) {
	db := setupTestDB(t)
	if err := events.MigrateOutbox(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	dir := t.TempDir()
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret, UploadDir: dir} // Outbox nil
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /media", h.Auth(h.UploadMedia))
	s := httptest.NewServer(mux)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass123!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass123!")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "pic.png")
	fw.Write(makeRealPNG())
	w.Close()

	req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if rows := pendingOutboxRows(t, db); len(rows) != 0 {
		t.Fatalf("expected 0 pending rows (no outbox wiring), got %d", len(rows))
	}
}
