package handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"omoikane-backend/internal/auth"
	"omoikane-backend/internal/audit"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"
)

type createPageRequest struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Content         string `json:"content"`
	MetaTitle       string `json:"metaTitle"`
	MetaDescription string `json:"metaDescription"`
	MetaKeywords    string `json:"metaKeywords"`
	Status          string `json:"status"`
	ParentID        *uint  `json:"parentId"`
	InMenu          *bool  `json:"inMenu"`
}

type updatePageRequest struct {
	Title           *string `json:"title,omitempty"`
	Slug            *string `json:"slug,omitempty"`
	Content         *string `json:"content,omitempty"`
	MetaTitle       *string `json:"metaTitle,omitempty"`
	MetaDescription *string `json:"metaDescription,omitempty"`
	MetaKeywords    *string `json:"metaKeywords,omitempty"`
	Status          *string `json:"status,omitempty"`
	ParentID        *uint   `json:"parentId,omitempty"`
	InMenu          *bool   `json:"inMenu,omitempty"`
}

type reorderPagesRequest struct {
	PageIds []uint `json:"pageIds"`
}

// GetPages returns pages ordered by sort_order. Public callers only see published pages;
// use ?menu=true to get only published pages flagged for the menu.
// @Summary List pages
// @Description Returns pages. Unauthenticated requests and ?menu=true only receive published pages.
// @Tags pages
// @Produce json
// @Param menu query bool false "Return only published menu pages"
// @Success 200 {array} map[string]interface{}
// @Router /pages [get]
func (h *Handler) GetPages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var pages []models.Page
	query := h.DB.Order("sort_order asc")

	// menu=true: return only published pages with inMenu enabled (for public menu widget)
	if r.URL.Query().Get("menu") == "true" {
		query = query.Where("status = ? AND in_menu = ?", "published", true)
	} else {
		// Only filter by published status for unauthenticated requests
		cookie, err := r.Cookie("session")
		var tokenStr string
		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		}
		claims, _ := auth.ValidateToken(tokenStr, h.JWTSecret)
		if claims == nil {
			query = query.Where("status = ?", "published")
		}
	}
	query.Find(&pages)

	result := make([]map[string]interface{}, 0)
	for _, p := range pages {
		result = append(result, sanitizePageJSON(p))
	}

	json.NewEncoder(w).Encode(result)
}

// GetPage returns a single page by ID.
// @Summary Get page by ID
// @Description Returns a single page by its numeric ID.
// @Tags pages
// @Produce json
// @Param id path int true "Page ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id} [get]
func (h *Handler) GetPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid page ID"})
		return
	}

	var page models.Page
	if err := h.DB.First(&page, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
		return
	}

	json.NewEncoder(w).Encode(sanitizePageJSON(page))
}

// GetPageBySlug returns a published page by slug.
// @Summary Get page by slug
// @Description Returns a single published page by its slug. Includes parent title/slug when a parent exists.
// @Tags pages
// @Produce json
// @Param slug path string true "Page slug"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /pages/slug/{slug} [get]
func (h *Handler) GetPageBySlug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	slug := r.PathValue("slug")

	var page models.Page
	if err := h.DB.Where("slug = ? AND status = ?", slug, "published").First(&page).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
		return
	}

	result := sanitizePageJSON(page)

	if page.ParentID != nil {
		var parent models.Page
		if err := h.DB.First(&parent, *page.ParentID).Error; err == nil {
			result["parentTitle"] = parent.Title
			result["parentSlug"] = parent.Slug
		}
	}

	json.NewEncoder(w).Encode(result)
}

// CreatePage creates a new page.
// @Summary Create page
// @Description Creates a page. Slug defaults to a generated slug from the title when omitted.
// @Tags pages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createPageRequest true "Page details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /pages [post]
func (h *Handler) CreatePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Title           string `json:"title"`
		Slug            string `json:"slug"`
		Content         string `json:"content"`
		MetaTitle       string `json:"metaTitle"`
		MetaDescription string `json:"metaDescription"`
		MetaKeywords    string `json:"metaKeywords"`
		Status          string `json:"status"`
		ParentID        *uint  `json:"parentId"`
		InMenu          *bool  `json:"inMenu"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Title is required"})
		return
	}
	if req.Slug == "" {
		req.Slug = generateSlug(req.Title)
	}

	// Check for duplicate slug
	var count int64
	h.DB.Model(&models.Page{}).Where("slug = ?", req.Slug).Count(&count)
	if count > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "A page with this slug already exists"})
		return
	}

	status := "draft"
	if req.Status == "published" {
		status = "published"
	}

	inMenu := false
	if req.InMenu != nil {
		inMenu = *req.InMenu
	}

	page := models.Page{
		Title:           req.Title,
		Slug:            req.Slug,
		Content:         req.Content,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		MetaKeywords:    req.MetaKeywords,
		ParentID:        req.ParentID,
		Status:          status,
		InMenu:          inMenu,
		PreviewToken:    generatePreviewToken(),
	}

	h.DB.Create(&page)

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "create",
		EntityType: "page",
		EntityID:   page.ID,
		Detail:     "Created page \"" + page.Title + "\"",
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizePageJSON(page))
}

// UpdatePage updates an existing page.
// @Summary Update page
// @Description Updates the provided fields of a page.
// @Tags pages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Page ID"
// @Param body body updatePageRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id} [put]
func (h *Handler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid page ID"})
		return
	}

	var page models.Page
	if err := h.DB.First(&page, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
		return
	}

	var req struct {
		Title           *string `json:"title,omitempty"`
		Slug            *string `json:"slug,omitempty"`
		Content         *string `json:"content,omitempty"`
		MetaTitle       *string `json:"metaTitle,omitempty"`
		MetaDescription *string `json:"metaDescription,omitempty"`
		MetaKeywords    *string `json:"metaKeywords,omitempty"`
		Status          *string `json:"status,omitempty"`
		ParentID        *uint   `json:"parentId,omitempty"`
		InMenu          *bool   `json:"inMenu,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Slug != nil {
		updates["slug"] = *req.Slug
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.MetaTitle != nil {
		updates["meta_title"] = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		updates["meta_description"] = *req.MetaDescription
	}
	if req.MetaKeywords != nil {
		updates["meta_keywords"] = *req.MetaKeywords
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.InMenu != nil {
		updates["in_menu"] = *req.InMenu
	}

	if len(updates) > 0 {
		h.DB.Model(&page).Updates(updates)
	}

	h.DB.First(&page, id)

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "update",
		EntityType: "page",
		EntityID:   page.ID,
		Detail:     "Updated page \"" + page.Title + "\"",
	})

	json.NewEncoder(w).Encode(sanitizePageJSON(page))
}

// DeletePage soft-deletes a page.
// @Summary Delete page
// @Description Soft-deletes a page; the record moves to trash and can be restored.
// @Tags pages
// @Produce json
// @Security BearerAuth
// @Param id path int true "Page ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id} [delete]
func (h *Handler) DeletePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid page ID"})
		return
	}

	var page models.Page
	if err := h.DB.First(&page, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
		return
	}

	h.DB.Delete(&page)

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "delete",
		EntityType: "page",
		EntityID:   page.ID,
		Detail:     "Deleted page \"" + page.Title + "\"",
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ReorderPages sets the sort order of pages.
// @Summary Reorder pages
// @Description Reorders pages by assigning sort_order based on the order of pageIds.
// @Tags pages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body reorderPagesRequest true "Ordered page IDs"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /pages/reorder [put]
func (h *Handler) ReorderPages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		PageIds []uint `json:"pageIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	for i, pageID := range req.PageIds {
		h.DB.Model(&models.Page{}).Where("id = ?", pageID).Update("sort_order", i)
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// BatchPages performs a bulk action on pages.
// @Summary Batch page actions
// @Description Applies an action (delete, publish, draft) to multiple pages.
// @Tags pages
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body batchRequest true "Action and page IDs"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /pages/batch [post]
func (h *Handler) BatchPages(w http.ResponseWriter, r *http.Request) {
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
		h.DB.Delete(&models.Page{}, req.IDs)
	case "publish":
		h.DB.Model(&models.Page{}).Where("id IN ?", req.IDs).Update("status", "published")
	case "draft":
		h.DB.Model(&models.Page{}).Where("id IN ?", req.IDs).Update("status", "draft")
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action"})
		return
	}

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "batch_" + req.Action,
		EntityType: "page",
		Detail:     "Batch " + req.Action + " on " + strconv.Itoa(len(req.IDs)) + " pages",
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func sanitizePageJSON(p models.Page) map[string]interface{} {
	return map[string]interface{}{
		"id":              p.ID,
		"title":           p.Title,
		"slug":            p.Slug,
		"content":         p.Content,
		"metaTitle":       p.MetaTitle,
		"metaDescription": p.MetaDescription,
		"metaKeywords":    p.MetaKeywords,
		"parentId":        p.ParentID,
		"sortOrder":       p.SortOrder,
		"status":          p.Status,
		"inMenu":          p.InMenu,
		"previewToken":    p.PreviewToken,
		"createdAt":       p.CreatedAt,
		"updatedAt":       p.UpdatedAt,
	}
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "--", "-")
	return slug
}

func generatePreviewToken() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 32)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}
