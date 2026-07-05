package handlers_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func mediaServer(db *gorm.DB, uploadDir string) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret, UploadDir: uploadDir}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /media", h.Auth(h.GetMedia))
	mux.HandleFunc("POST /media", h.Auth(h.UploadMedia))
	mux.HandleFunc("GET /media/{id}", h.Auth(h.GetMediaItem))
	mux.HandleFunc("DELETE /media/{id}", h.Auth(h.DeleteMedia))
	return httptest.NewServer(mux)
}

func TestUploadMedia_AuthRequired(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	s := mediaServer(db, dir)
	defer s.Close()

	resp, err := http.Post(s.URL+"/media", "multipart/form-data", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestUploadMedia_UploadsFile(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	s := mediaServer(db, dir)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fileWriter, _ := w.CreateFormFile("file", "test.txt")
	fileWriter.Write([]byte("hello world"))
	w.Close()

	req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	data := decodeJSON(t, readBody(t, resp))
	media, ok := data["media"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'media' object in response")
	}
	if media["filename"] != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %v", media["filename"])
	}
	if media["mimeType"] != "text/plain; charset=utf-8" {
		t.Errorf("Expected mimeType 'text/plain; charset=utf-8', got %v", media["mimeType"])
	}
	if media["size"] != float64(11) {
		t.Errorf("Expected size 11, got %v", media["size"])
	}
	dataStr, ok := media["data"].(string)
	if !ok || dataStr == "" {
		t.Error("Expected non-empty data field")
	}
	if !strings.HasPrefix(dataStr, "data:text/plain") {
		t.Error("Expected data URI prefix")
	}

	// Verify file exists on disk
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Errorf("Expected 1 file on disk, got %d", len(files))
	}
}

func TestGetMedia_ListsItems(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	s := mediaServer(db, dir)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	// Upload two files
	for _, name := range []string{"a.txt", "b.txt"} {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("file", name)
		fw.Write([]byte("content"))
		w.Close()

		req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.AddCookie(cookie)
		http.DefaultClient.Do(req)
	}

	resp := authenticatedRequest(t, "GET", s.URL+"/media", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	items := decodeJSONArray(t, readBody(t, resp))
	if len(items) != 2 {
		t.Errorf("Expected 2 media items, got %d", len(items))
	}
}

func TestDeleteMedia_DeletesFromDisk(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	s := mediaServer(db, dir)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "delete-me.txt")
	fw.Write([]byte("to be deleted"))
	w.Close()

	req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)
	resp, _ := http.DefaultClient.Do(req)
	data := decodeJSON(t, readBody(t, resp))
	mediaItem := data["media"].(map[string]interface{})

	mediaID := int(mediaItem["id"].(float64))
	_ = mediaID
	resp.Body.Close()

	delResp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/media/%d", s.URL, mediaID), "", cookie)
	defer delResp.Body.Close()

	if delResp.StatusCode != 200 {
		t.Errorf("Expected 200 on delete, got %d", delResp.StatusCode)
	}

	// File remains on disk after soft-delete (hard-purge only via trash)
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Errorf("Expected 1 file on disk after soft-delete, got %d", len(files))
	}

	// DB record should have deleted_at set (soft-deleted)
	var count int64
	db.Model(&models.MediaItem{}).Unscoped().Where("id = ? AND deleted_at IS NOT NULL", mediaID).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 soft-deleted media record, got %d", count)
	}
}

func TestUploadMedia_ImageDetectsMime(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	s := mediaServer(db, dir)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "Pass1234!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "Pass1234!")

	// Minimal PNG (1x1 pixel)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG header
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0x60, 0x60, 0x00, 0x00,
		0x00, 0x04, 0x00, 0x01, 0x27, 0x34, 0x27, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
		0x42, 0x60, 0x82,
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "image.png")
	fw.Write(pngData)
	w.Close()

	req, _ := http.NewRequest("POST", s.URL+"/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data := decodeJSON(t, readBody(t, resp))
	mediaItem := data["media"].(map[string]interface{})
	if mediaItem["mimeType"] != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got %v", mediaItem["mimeType"])
	}
}
