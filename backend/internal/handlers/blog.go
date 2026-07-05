package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid tag ID"})
		return
	}

	if err := h.DB.Delete(&models.Tag{}, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Tag not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid category ID"})
		return
	}

	if err := h.DB.Delete(&models.Category{}, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Category not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func lookupUserName(h *Handler, userID uint) string {
	if h == nil {
		return ""
	}
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return ""
	}
	return user.Name
}

func (h *Handler) GetPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var posts []models.BlogPost
	h.DB.Where("status = ?", "published").Order("created_at desc").Find(&posts)

	result := make([]map[string]interface{}, 0)
	for _, p := range posts {
		result = append(result, sanitizePostJSON(h, p))
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetAdminPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var posts []models.BlogPost
	h.DB.Order("created_at desc").Find(&posts)

	result := make([]map[string]interface{}, 0)
	for _, p := range posts {
		result = append(result, sanitizePostJSON(h, p))
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid post ID"})
		return
	}

	var post models.BlogPost
	if err := h.DB.First(&post, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Post not found"})
		return
	}

	json.NewEncoder(w).Encode(sanitizePostJSON(h, post))
}

func (h *Handler) GetPostBySlug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	slug := r.PathValue("slug")

	var post models.BlogPost
	if err := h.DB.Where("slug = ? AND status = ?", slug, "published").First(&post).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Post not found"})
		return
	}

	json.NewEncoder(w).Encode(sanitizePostJSON(h, post))
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Title         string   `json:"title"`
		Slug          string   `json:"slug"`
		Content       string   `json:"content"`
		Excerpt       string   `json:"excerpt"`
		Status        string   `json:"status"`
		FeaturedImage string   `json:"featuredImage"`
		Tags          []string `json:"tags"`
		CategoryID    *uint    `json:"categoryId"`
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

	status := "draft"
	if req.Status == "published" {
		status = "published"
	}

	userID := middleware.GetUserID(r)
	post := models.BlogPost{
		Title:         req.Title,
		Slug:          req.Slug,
		Content:       req.Content,
		Excerpt:       req.Excerpt,
		AuthorID:      userID,
		Status:        status,
		FeaturedImage: req.FeaturedImage,
		CategoryID:    req.CategoryID,
	}

	h.DB.Create(&post)

	// Associate tags
	for _, tagName := range req.Tags {
		var tag models.Tag
		if err := h.DB.Where("name = ?", tagName).First(&tag).Error; err == nil {
			h.DB.Model(&post).Association("Tags").Append(&tag)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizePostJSON(h, post))
}

func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid post ID"})
		return
	}

	var post models.BlogPost
	if err := h.DB.First(&post, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Post not found"})
		return
	}

	var req struct {
		Title         *string `json:"title,omitempty"`
		Slug          *string `json:"slug,omitempty"`
		Content       *string `json:"content,omitempty"`
		Excerpt       *string `json:"excerpt,omitempty"`
		Status        *string `json:"status,omitempty"`
		FeaturedImage *string `json:"featuredImage,omitempty"`
		CategoryID    *uint   `json:"categoryId,omitempty"`
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
	if req.Excerpt != nil {
		updates["excerpt"] = *req.Excerpt
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.FeaturedImage != nil {
		updates["featured_image"] = *req.FeaturedImage
	}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}

	if len(updates) > 0 {
		h.DB.Model(&post).Updates(updates)
	}

	h.DB.First(&post, id)
	json.NewEncoder(w).Encode(sanitizePostJSON(h, post))
}

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid post ID"})
		return
	}

	var post models.BlogPost
	if err := h.DB.First(&post, id).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Post not found"})
		return
	}

	h.DB.Delete(&post)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) BatchPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Action string   `json:"action"`
		IDs    []string `json:"ids"`
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
		h.DB.Delete(&models.BlogPost{}, req.IDs)
	case "publish":
		h.DB.Model(&models.BlogPost{}).Where("id IN ?", req.IDs).Update("status", "published")
	case "draft":
		h.DB.Model(&models.BlogPost{}).Where("id IN ?", req.IDs).Update("status", "draft")
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action"})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	postID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid post ID"})
		return
	}

	var post models.BlogPost
	if err := h.DB.First(&post, postID).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Post not found"})
		return
	}

	userID := middleware.GetUserID(r)

	var like models.Like
	result := h.DB.Where("blog_post_id = ? AND user_id = ?", postID, userID).First(&like)

	liked := false
	if result.Error != nil {
		like = models.Like{BlogPostID: uint(postID), UserID: userID}
		h.DB.Create(&like)
		h.DB.Model(&post).UpdateColumn("like_count", gorm.Expr("like_count + 1"))
		liked = true
	} else {
		h.DB.Delete(&like)
		h.DB.Model(&post).UpdateColumn("like_count", gorm.Expr("like_count - 1"))
	}

	h.DB.First(&post, postID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"liked":  liked,
		"count":  post.LikeCount,
	})
}

func sanitizeTag(t models.Tag) map[string]interface{} {
	return map[string]interface{}{
		"id":   t.ID,
		"name": t.Name,
		"slug": t.Slug,
	}
}

func (h *Handler) GetTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var tags []models.Tag
	h.DB.Find(&tags)

	result := make([]map[string]interface{}, 0)
	for _, t := range tags {
		result = append(result, sanitizeTag(t))
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name is required"})
		return
	}
	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	}

	tag := models.Tag{Name: req.Name, Slug: req.Slug}
	h.DB.Create(&tag)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   tag.ID,
		"name": tag.Name,
		"slug": tag.Slug,
	})
}

func sanitizeCategory(c models.Category) map[string]interface{} {
	return map[string]interface{}{
		"id":          c.ID,
		"name":        c.Name,
		"slug":        c.Slug,
		"description": c.Description,
	}
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var categories []models.Category
	h.DB.Find(&categories)

	result := make([]map[string]interface{}, 0)
	for _, c := range categories {
		result = append(result, sanitizeCategory(c))
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name is required"})
		return
	}
	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	}

	cat := models.Category{Name: req.Name, Slug: req.Slug, Description: req.Description}
	h.DB.Create(&cat)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          cat.ID,
		"name":        cat.Name,
		"slug":        cat.Slug,
		"description": cat.Description,
	})
}

func sanitizePostJSON(h *Handler, p models.BlogPost) map[string]interface{} {
	var tags []models.Tag
	if h != nil {
		h.DB.Model(&p).Association("Tags").Find(&tags)
	}

	tagNames := make([]string, 0)
	for _, t := range tags {
		tagNames = append(tagNames, t.Name)
	}

	var categoryID interface{}
	if p.CategoryID != nil {
		categoryID = *p.CategoryID
	}

	authorName := lookupUserName(h, p.AuthorID)

	return map[string]interface{}{
		"id":            p.ID,
		"title":         p.Title,
		"slug":          p.Slug,
		"content":       p.Content,
		"excerpt":       p.Excerpt,
		"authorId":      p.AuthorID,
		"authorName":    authorName,
		"status":        p.Status,
		"publishDate":   p.PublishDate,
		"featuredImage": p.FeaturedImage,
		"likeCount":     p.LikeCount,
		"tags":          tagNames,
		"categoryId":    categoryID,
		"createdAt":     p.CreatedAt,
		"updatedAt":     p.UpdatedAt,
	}
}
