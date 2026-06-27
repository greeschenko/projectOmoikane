package handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"omoikane-backend/internal/models"
)

func (h *Handler) GetPages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var pages []models.Page
	h.DB.Where("status = ?", "published").Order("sort_order asc").Find(&pages)

	result := make([]map[string]interface{}, 0)
	for _, p := range pages {
		result = append(result, sanitizePageJSON(p))
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"pages": result,
	})
}

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

func (h *Handler) GetPageBySlug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	slug := r.PathValue("slug")

	var page models.Page
	if err := h.DB.Where("slug = ? AND status = ?", slug, "published").First(&page).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Page not found"})
		return
	}

	json.NewEncoder(w).Encode(sanitizePageJSON(page))
}

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

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizePageJSON(page))
}

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
	json.NewEncoder(w).Encode(sanitizePageJSON(page))
}

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
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

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
