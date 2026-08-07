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

// UploadMedia uploads a file via multipart form (field "file").
// @Summary Upload media file
// @Description Uploads a file (multipart/form-data, field name "file"). Returns the media item with base64-encoded data.
// @Tags media
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File to upload"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /media [post]
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

// GetMedia returns all media items with base64-encoded file data.
// @Summary List media
// @Description Returns all media items, newest first, each with base64-encoded data.
// @Tags media
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{}
// @Router /media [get]
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

// GetMediaItem returns a single media item by ID.
// @Summary Get media item
// @Description Returns a single media item by its numeric ID.
// @Tags media
// @Produce json
// @Security BearerAuth
// @Param id path int true "Media ID"
// @Success 200 {object} models.MediaItem
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /media/{id} [get]
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

// DeleteMedia soft-deletes a media item.
// @Summary Delete media item
// @Description Soft-deletes a media record; the file stays on disk until hard-purged from trash.
// @Tags media
// @Produce json
// @Security BearerAuth
// @Param id path int true "Media ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /media/{id} [delete]
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

	// Soft-delete only — file stays on disk until hard-purge from trash
	h.DB.Delete(&item)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// BatchMedia performs a bulk action on media items.
// @Summary Batch media actions
// @Description Applies an action (delete) to multiple media items.
// @Tags media
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body batchRequest true "Action and media IDs"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /media/batch [post]
func (h *Handler) BatchMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Action string `json:"action"`
		IDs    []uint `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No IDs provided"})
		return
	}

	switch req.Action {
	case "delete":
		h.DB.Delete(&models.MediaItem{}, req.IDs)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func generateUniqueName() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
