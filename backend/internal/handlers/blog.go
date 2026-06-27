package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func (h *Handler) GetPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var posts []models.BlogPost
	h.DB.Where("status = ?", "published").Order("created_at desc").Find(&posts)

	result := make([]map[string]interface{}, 0)
	for _, p := range posts {
		result = append(result, sanitizePostJSON(p))
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": result,
	})
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

	json.NewEncoder(w).Encode(sanitizePostJSON(post))
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

	json.NewEncoder(w).Encode(sanitizePostJSON(post))
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Title         string `json:"title"`
		Slug          string `json:"slug"`
		Content       string `json:"content"`
		Excerpt       string `json:"excerpt"`
		Status        string `json:"status"`
		FeaturedImage string `json:"featuredImage"`
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
	}

	h.DB.Create(&post)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizePostJSON(post))
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

	if len(updates) > 0 {
		h.DB.Model(&post).Updates(updates)
	}

	h.DB.First(&post, id)
	json.NewEncoder(w).Encode(sanitizePostJSON(post))
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
		"liked":     liked,
		"likeCount": post.LikeCount,
	})
}

func (h *Handler) GetTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var tags []models.Tag
	h.DB.Find(&tags)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tags": tags,
	})
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

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var categories []models.Category
	h.DB.Find(&categories)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categories,
	})
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

func sanitizePostJSON(p models.BlogPost) map[string]interface{} {
	return map[string]interface{}{
		"id":            p.ID,
		"title":         p.Title,
		"slug":          p.Slug,
		"content":       p.Content,
		"excerpt":       p.Excerpt,
		"authorId":      p.AuthorID,
		"status":        p.Status,
		"publishDate":   p.PublishDate,
		"featuredImage": p.FeaturedImage,
		"likeCount":     p.LikeCount,
		"createdAt":     p.CreatedAt,
		"updatedAt":     p.UpdatedAt,
	}
}
