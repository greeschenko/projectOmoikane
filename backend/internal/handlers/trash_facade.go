package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// trashOwners maps each trash entity to the owning service key used by the
// facade's URL routing.
var trashOwners = map[string]string{
	"user":     "auth",
	"page":     "content",
	"post":     "content",
	"tag":      "content",
	"category": "content",
	"media":    "media",
	"contact":  "messages",
	"message":  "messages",
}

// TrashFacade is the cross-cutting aggregator (Phase 31 §2). It owns no data
// layer: it fans out each trash operation to the owning services' internal
// /internal/trash* endpoints, which are authenticated with the shared internal
// token (X-Internal-Token header).
type TrashFacade struct {
	AuthURL, ContentURL, MediaURL, MessagesURL string
	InternalToken                              string
	Client                                     *http.Client
}

func (f *TrashFacade) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// allServiceURLs returns the configured owning-service base URLs in canonical
// order (skipping unset ones, handy for tests).
func (f *TrashFacade) allServiceURLs() []string {
	urls := make([]string, 0, 4)
	for _, u := range []string{f.AuthURL, f.ContentURL, f.MediaURL, f.MessagesURL} {
		if u != "" {
			urls = append(urls, u)
		}
	}
	return urls
}

// ownerURL returns the owning-service base URL for an entity, or "" when the
// entity is unknown or its owner is not configured.
func (f *TrashFacade) ownerURL(entity string) string {
	if owner, ok := trashOwners[entity]; ok {
		switch owner {
		case "auth":
			return f.AuthURL
		case "content":
			return f.ContentURL
		case "media":
			return f.MediaURL
		case "messages":
			return f.MessagesURL
		}
	}
	return ""
}

func (f *TrashFacade) internalRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if f.InternalToken != "" {
		req.Header.Set("X-Internal-Token", f.InternalToken)
	}
	return req, nil
}

func writeBadGateway(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]string{"error": "Trash service unavailable"})
}

// GetTrash aggregates the soft-deleted rows of every owning service (admin only).
// @Summary List trash items
// @Description Returns soft-deleted pages, users, posts, media, contacts, messages, tags and categories.
// @Tags trash
// @Produce json
// @Security BearerAuth
// @Success 200 {array} TrashItem
// @Router /trash [get]
func (f *TrashFacade) GetTrash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	items := []TrashItem{}
	for _, base := range f.allServiceURLs() {
		req, err := f.internalRequest(http.MethodGet, base+"/internal/trash", nil)
		if err != nil {
			writeBadGateway(w)
			return
		}
		resp, err := f.client().Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			writeBadGateway(w)
			return
		}
		var part []TrashItem
		json.NewDecoder(resp.Body).Decode(&part)
		resp.Body.Close()
		items = append(items, part...)
	}

	json.NewEncoder(w).Encode(items)
}

// GetTrashCount sums every owning service's trash total (admin only).
// @Summary Trash count
// @Description Returns the total number of soft-deleted items across all entities.
// @Tags trash
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]int64
// @Router /trash/count [get]
func (f *TrashFacade) GetTrashCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var total int64
	for _, base := range f.allServiceURLs() {
		req, err := f.internalRequest(http.MethodGet, base+"/internal/trash/count", nil)
		if err != nil {
			writeBadGateway(w)
			return
		}
		resp, err := f.client().Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			writeBadGateway(w)
			return
		}
		var out struct {
			Count int64 `json:"count"`
		}
		json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		total += out.Count
	}

	json.NewEncoder(w).Encode(map[string]int64{"count": total})
}

// RestoreItem routes a restore to the entity's owning service (admin only).
// @Summary Restore trash item
// @Description Restores a soft-deleted item by entity type and ID.
// @Tags trash
// @Produce json
// @Security BearerAuth
// @Param entity path string true "Entity type (page, user, post, media, contact, message, tag, category)"
// @Param id path int true "Item ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /trash/{entity}/{id}/restore [post]
func (f *TrashFacade) RestoreItem(w http.ResponseWriter, r *http.Request) {
	entity := r.PathValue("entity")
	base := f.ownerURL(entity)
	if base == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
		return
	}
	f.routeAction(w, http.MethodPost, base+"/internal/trash/"+entity+"/"+r.PathValue("id")+"/restore")
}

// HardDeleteItem routes a permanent delete to the entity's owning service
// (admin only). Media disk cleanup happens in the media service.
// @Summary Permanently delete trash item
// @Description Hard-deletes an item by entity type and ID. Media files are removed from disk.
// @Tags trash
// @Produce json
// @Security BearerAuth
// @Param entity path string true "Entity type (page, user, post, media, contact, message, tag, category)"
// @Param id path int true "Item ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /trash/{entity}/{id} [delete]
func (f *TrashFacade) HardDeleteItem(w http.ResponseWriter, r *http.Request) {
	entity := r.PathValue("entity")
	base := f.ownerURL(entity)
	if base == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
		return
	}
	f.routeAction(w, http.MethodDelete, base+"/internal/trash/"+entity+"/"+r.PathValue("id"))
}

// EmptyTrash empties all (or one ?entity's) trash across the owning services
// (admin only).
// @Summary Empty trash
// @Description Hard-deletes all trash items, or only the given entity's items when ?entity= is provided.
// @Tags trash
// @Produce json
// @Security BearerAuth
// @Param entity query string false "Only empty this entity type (page, user, post, media, contact, message, tag, category)"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /trash [delete]
func (f *TrashFacade) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entity := r.URL.Query().Get("entity")
	if entity != "" {
		base := f.ownerURL(entity)
		if base == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unknown entity type"})
			return
		}
		f.routeAction(w, http.MethodDelete, base+"/internal/trash?entity="+entity)
		return
	}

	for _, base := range f.allServiceURLs() {
		req, err := f.internalRequest(http.MethodDelete, base+"/internal/trash", nil)
		if err != nil {
			writeBadGateway(w)
			return
		}
		resp, err := f.client().Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			writeBadGateway(w)
			return
		}
		resp.Body.Close()
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// routeAction forwards a restore/hard-delete to the owning service and reflects
// its response (status + body) back to the caller.
func (f *TrashFacade) routeAction(w http.ResponseWriter, method, url string) {
	req, err := f.internalRequest(method, url, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "Trash service unavailable"})
		return
	}
	resp, err := f.client().Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "Trash service unavailable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
