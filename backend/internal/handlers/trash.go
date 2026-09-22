package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"omoikane-backend/internal/models"
)

// TrashItem is one soft-deleted row surfaced by the owning services' internal
// trash endpoints and aggregated by the trash service.
type TrashItem struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Entity    string    `json:"entity"`
	DeletedAt time.Time `json:"deletedAt"`
}

// trashEntityOwned reports whether this service owns the entity's trash surface
// (i.e. the entity appears in the handler's TrashEntities).
func (h *Handler) trashEntityOwned(entity string) bool {
	for _, e := range h.TrashEntities {
		if e == entity {
			return true
		}
	}
	return false
}

// appendTrashItems appends the soft-deleted rows for one owned entity.
func (h *Handler) appendTrashItems(entity string, items *[]TrashItem) {
	switch entity {
	case "page":
		var rows []models.Page
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Title, Entity: "page", DeletedAt: r.DeletedAt.Time})
		}
	case "user":
		var rows []models.User
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Name, Entity: "user", DeletedAt: r.DeletedAt.Time})
		}
	case "post":
		var rows []models.BlogPost
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Title, Entity: "post", DeletedAt: r.DeletedAt.Time})
		}
	case "media":
		var rows []models.MediaItem
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Filename, Entity: "media", DeletedAt: r.DeletedAt.Time})
		}
	case "contact":
		var rows []models.ContactMessage
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Subject, Entity: "contact", DeletedAt: r.DeletedAt.Time})
		}
	case "message":
		var rows []models.Message
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Title, Entity: "message", DeletedAt: r.DeletedAt.Time})
		}
	case "tag":
		var rows []models.Tag
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Name, Entity: "tag", DeletedAt: r.DeletedAt.Time})
		}
	case "category":
		var rows []models.Category
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Order("deleted_at desc").Find(&rows)
		for _, r := range rows {
			*items = append(*items, TrashItem{ID: r.ID, Title: r.Name, Entity: "category", DeletedAt: r.DeletedAt.Time})
		}
	}
}

// trashRowCount returns the number of soft-deleted rows for one owned entity.
func (h *Handler) trashRowCount(entity string) int64 {
	var c int64
	switch entity {
	case "page":
		h.DB.Unscoped().Model(&models.Page{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "user":
		h.DB.Unscoped().Model(&models.User{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "post":
		h.DB.Unscoped().Model(&models.BlogPost{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "media":
		h.DB.Unscoped().Model(&models.MediaItem{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "contact":
		h.DB.Unscoped().Model(&models.ContactMessage{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "message":
		h.DB.Unscoped().Model(&models.Message{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "tag":
		h.DB.Unscoped().Model(&models.Tag{}).Where("deleted_at IS NOT NULL").Count(&c)
	case "category":
		h.DB.Unscoped().Model(&models.Category{}).Where("deleted_at IS NOT NULL").Count(&c)
	}
	return c
}

// restoreTrashRow clears the soft-delete marker for a row owned by this service.
func (h *Handler) restoreTrashRow(entity string, id uint) {
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
	}
}

// hardDeleteTrashRow permanently deletes a soft-deleted row. Media rows also
// remove their files from disk (media service owns that cleanup).
func (h *Handler) hardDeleteTrashRow(entity string, id uint) {
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
	}
}

// emptyTrashRows hard-deletes all soft-deleted rows of one owned entity. Media
// files are removed from disk when the entity is media.
func (h *Handler) emptyTrashRows(entity string) {
	switch entity {
	case "page":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Page{})
	case "user":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.User{})
	case "post":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.BlogPost{})
	case "media":
		var items []models.MediaItem
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Find(&items)
		for _, m := range items {
			osRemove(m.FilePath)
		}
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.MediaItem{})
	case "contact":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.ContactMessage{})
	case "message":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Message{})
	case "tag":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Tag{})
	case "category":
		h.DB.Unscoped().Where("deleted_at IS NOT NULL").Delete(&models.Category{})
	}
}

// InternalTrashList returns this service's soft-deleted rows (internal). The
// trash service fans out to each owning service and merges the results.
// @Summary List owned trash items (internal)
// @Description Returns the soft-deleted rows for the entities this service owns.
// @Tags trash
// @Produce json
// @Success 200 {array} TrashItem
// @Router /internal/trash [get]
func (h *Handler) InternalTrashList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	items := []TrashItem{}
	for _, entity := range h.TrashEntities {
		h.appendTrashItems(entity, &items)
	}

	json.NewEncoder(w).Encode(items)
}

// InternalTrashCount returns this service's trash total (internal).
// @Summary Count owned trash (internal)
// @Description Returns the total number of soft-deleted rows this service owns.
// @Tags trash
// @Produce json
// @Success 200 {object} map[string]int64
// @Router /internal/trash/count [get]
func (h *Handler) InternalTrashCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var count int64
	for _, entity := range h.TrashEntities {
		count += h.trashRowCount(entity)
	}

	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

// InternalTrashRestore restores one owned soft-deleted row (internal).
// @Summary Restore owned trash item (internal)
// @Description Restores a soft-deleted row by entity type and ID.
// @Tags trash
// @Produce json
// @Param entity path string true "Owned entity type (page, user, post, media, contact, message, tag, category)"
// @Param id path int true "Item ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /internal/trash/{entity}/{id}/restore [post]
func (h *Handler) InternalTrashRestore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.PathValue("entity")
	if !h.trashEntityOwned(entity) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid ID"})
		return
	}

	h.restoreTrashRow(entity, uint(id))

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
	h.flushCache()
}

// InternalTrashHardDelete permanently deletes one owned soft-deleted row (internal).
// @Summary Permanently delete owned trash item (internal)
// @Description Hard-deletes a soft-deleted row by entity type and ID; media files are removed from disk.
// @Tags trash
// @Produce json
// @Param entity path string true "Owned entity type (page, user, post, media, contact, message, tag, category)"
// @Param id path int true "Item ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /internal/trash/{entity}/{id} [delete]
func (h *Handler) InternalTrashHardDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.PathValue("entity")
	if !h.trashEntityOwned(entity) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid ID"})
		return
	}

	h.hardDeleteTrashRow(entity, uint(id))

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
	h.flushCache()
}

// InternalTrashEmpty empties all (or one ?entity's) owned trash (internal).
// @Summary Empty owned trash (internal)
// @Description Hard-deletes all owned trash rows, or only the given entity's when ?entity= is provided.
// @Tags trash
// @Produce json
// @Param entity query string false "Only empty this owned entity type"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /internal/trash [delete]
func (h *Handler) InternalTrashEmpty(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.URL.Query().Get("entity")
	if entity != "" {
		if !h.trashEntityOwned(entity) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
			return
		}
		h.emptyTrashRows(entity)
	} else {
		for _, e := range h.TrashEntities {
			h.emptyTrashRows(e)
		}
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
	h.flushCache()
}

var osRemove = func(path string) error {
	return os.Remove(path)
}
