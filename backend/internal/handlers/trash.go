package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"omoikane-backend/internal/models"
)

type TrashItem struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Entity    string    `json:"entity"`
	DeletedAt time.Time `json:"deletedAt"`
}

func (h *Handler) GetTrash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	items := []TrashItem{}

	var pages []models.Page
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&pages)
	for _, p := range pages {
		items = append(items, TrashItem{
			ID: p.ID, Title: p.Title, Entity: "page",
			DeletedAt: p.DeletedAt.Time,
		})
	}

	var users []models.User
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&users)
	for _, u := range users {
		items = append(items, TrashItem{
			ID: u.ID, Title: u.Name, Entity: "user",
			DeletedAt: u.DeletedAt.Time,
		})
	}

	var posts []models.BlogPost
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&posts)
	for _, p := range posts {
		items = append(items, TrashItem{
			ID: p.ID, Title: p.Title, Entity: "post",
			DeletedAt: p.DeletedAt.Time,
		})
	}

	var media []models.MediaItem
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&media)
	for _, m := range media {
		items = append(items, TrashItem{
			ID: m.ID, Title: m.Filename, Entity: "media",
			DeletedAt: m.DeletedAt.Time,
		})
	}

	var contacts []models.ContactMessage
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&contacts)
	for _, c := range contacts {
		items = append(items, TrashItem{
			ID: c.ID, Title: c.Subject, Entity: "contact",
			DeletedAt: c.DeletedAt.Time,
		})
	}

	var messages []models.Message
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&messages)
	for _, m := range messages {
		items = append(items, TrashItem{
			ID: m.ID, Title: m.Title, Entity: "message",
			DeletedAt: m.DeletedAt.Time,
		})
	}

	var tags []models.Tag
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&tags)
	for _, t := range tags {
		items = append(items, TrashItem{
			ID: t.ID, Title: t.Name, Entity: "tag",
			DeletedAt: t.DeletedAt.Time,
		})
	}

	var categories []models.Category
	h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&categories)
	for _, c := range categories {
		items = append(items, TrashItem{
			ID: c.ID, Title: c.Name, Entity: "category",
			DeletedAt: c.DeletedAt.Time,
		})
	}

	json.NewEncoder(w).Encode(items)
}

func (h *Handler) GetTrashCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var count int64

	for _, model := range []interface{}{&models.Page{}, &models.User{}, &models.BlogPost{},
		&models.MediaItem{}, &models.ContactMessage{}, &models.Message{},
		&models.Tag{}, &models.Category{}} {
		var c int64
		h.DB.Unscoped().Model(model).Where("deleted_at IS NOT NULL").Count(&c)
		count += c
	}

	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

func (h *Handler) RestoreItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.PathValue("entity")
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid ID"})
		return
	}

	switch entity {
	case "page":
		h.DB.Unscoped().Model(&models.Page{}).Where("id = ?", id).Update("deleted_at", nil)
	case "user":
		h.DB.Unscoped().Model(&models.User{}).Where("id = ?", id).Update("deleted_at", nil)
	case "post":
		h.DB.Unscoped().Model(&models.BlogPost{}).Where("id = ?", id).Update("deleted_at", nil)
	case "media":
		h.DB.Unscoped().Model(&models.MediaItem{}).Where("id = ?", id).Update("deleted_at", nil)
	case "contact":
		h.DB.Unscoped().Model(&models.ContactMessage{}).Where("id = ?", id).Update("deleted_at", nil)
	case "message":
		h.DB.Unscoped().Model(&models.Message{}).Where("id = ?", id).Update("deleted_at", nil)
	case "tag":
		h.DB.Unscoped().Model(&models.Tag{}).Where("id = ?", id).Update("deleted_at", nil)
	case "category":
		h.DB.Unscoped().Model(&models.Category{}).Where("id = ?", id).Update("deleted_at", nil)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) HardDeleteItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.PathValue("entity")
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid ID"})
		return
	}

	switch entity {
	case "page":
		h.DB.Unscoped().Delete(&models.Page{}, id)
	case "user":
		h.DB.Unscoped().Delete(&models.User{}, id)
	case "post":
		h.DB.Unscoped().Delete(&models.BlogPost{}, id)
	case "media":
		var item models.MediaItem
		if err := h.DB.Unscoped().First(&item, id).Error; err == nil {
			osRemove(item.FilePath)
			h.DB.Unscoped().Delete(&item)
		}
	case "contact":
		h.DB.Unscoped().Delete(&models.ContactMessage{}, id)
	case "message":
		h.DB.Unscoped().Delete(&models.Message{}, id)
	case "tag":
		h.DB.Unscoped().Delete(&models.Tag{}, id)
	case "category":
		h.DB.Unscoped().Delete(&models.Category{}, id)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.URL.Query().Get("entity")

	if entity == "" || entity == "media" {
		var media []models.MediaItem
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Find(&media)
		for _, m := range media {
			osRemove(m.FilePath)
		}
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.MediaItem{})
	}
	if entity == "" || entity == "page" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Page{})
	}
	if entity == "" || entity == "user" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.User{})
	}
	if entity == "" || entity == "post" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.BlogPost{})
	}
	if entity == "" || entity == "contact" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.ContactMessage{})
	}
	if entity == "" || entity == "message" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Message{})
	}
	if entity == "" || entity == "tag" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Tag{})
	}
	if entity == "" || entity == "category" {
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Category{})
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

var osRemove = func(path string) error {
	return os.Remove(path)
}
