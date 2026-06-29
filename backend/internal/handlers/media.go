package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"omoikane-backend/internal/models"
)

func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	r.ParseMultipartForm(32 << 20) // 32MB max

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file provided"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read file"})
		return
	}

	mimeType := http.DetectContentType(data)

	// Generate unique filename
	uniqueName := generateUniqueName() + filepath.Ext(header.Filename)

	// Ensure upload dir exists
	os.MkdirAll(h.UploadDir, 0755)

	// Save to disk
	dst := filepath.Join(h.UploadDir, uniqueName)
	if err := os.WriteFile(dst, data, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
		return
	}

	item := models.MediaItem{
		Filename: header.Filename,
		MimeType: mimeType,
		Size:     int64(len(data)),
		FilePath: dst,
	}
	h.DB.Create(&item)

	encoded := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"media": map[string]interface{}{
			"id":        item.ID,
			"filename":  item.Filename,
			"mimeType":  item.MimeType,
			"size":      item.Size,
			"data":      encoded,
			"createdAt": item.CreatedAt,
		},
	})
}

func readFileBase64(mimeType, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func (h *Handler) GetMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var items []models.MediaItem
	h.DB.Order("created_at desc").Find(&items)

	result := make([]map[string]interface{}, 0)
	for _, item := range items {
		result = append(result, map[string]interface{}{
			"id":        item.ID,
			"filename":  item.Filename,
			"mimeType":  item.MimeType,
			"size":      item.Size,
			"data":      readFileBase64(item.MimeType, item.FilePath),
			"createdAt": item.CreatedAt,
		})
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetMediaItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid media ID"})
		return
	}

	var item models.MediaItem
	if err := h.DB.First(&item, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Media not found"})
		return
	}

	json.NewEncoder(w).Encode(item)
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid media ID"})
		return
	}

	var item models.MediaItem
	if err := h.DB.First(&item, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Media not found"})
		return
	}

	// Delete from disk
	os.Remove(item.FilePath)

	h.DB.Delete(&item)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func generateUniqueName() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
