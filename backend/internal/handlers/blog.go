package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"omoikane-backend/internal/audit"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

type createPostRequest struct {
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	Content       string   `json:"content"`
	Excerpt       string   `json:"excerpt"`
	Status        string   `json:"status"`
	FeaturedImage string   `json:"featuredImage"`
	Tags          []string `json:"tags"`
	CategoryID    *uint    `json:"categoryId"`
}

type updatePostRequest struct {
	Title         *string  `json:"title,omitempty"`
	Slug          *string  `json:"slug,omitempty"`
	Content       *string  `json:"content,omitempty"`
	Excerpt       *string  `json:"excerpt,omitempty"`
	Status        *string  `json:"status,omitempty"`
	FeaturedImage *string  `json:"featuredImage,omitempty"`
	CategoryID    *uint    `json:"categoryId,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

type createTagRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type createCategoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// DeleteTag deletes a blog tag (admin only).
// @Summary Delete tag
// @Description Soft-deletes a blog tag.
// @Tags blog
// @Produce json
// @Security BearerAuth
// @Param id path int true "Tag ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blog/tags/{id} [delete]
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
	h.flushCache()
}

// DeleteCategory deletes a blog category (admin only).
// @Summary Delete category
// @Description Soft-deletes a blog category.
// @Tags blog
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blog/categories/{id} [delete]
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
	h.flushCache()
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

// GetPosts returns published blog posts (public).
// @Summary List published posts
// @Description Returns blog posts with status published, newest first.
// @Tags blog
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /blog/posts [get]
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

// GetAdminPosts returns all posts including drafts (admin only).
// @Summary List all posts
// @Description Returns all blog posts regardless of status, newest first.
// @Tags blog
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{}
// @Router /admin/blog/posts [get]
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

// GetPost returns a single blog post by ID.
// @Summary Get post by ID
// @Description Returns a single blog post by its numeric ID.
// @Tags blog
// @Produce json
// @Param id path int true "Post ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blog/posts/{id} [get]
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

// GetPostBySlug returns a published post by slug (public).
// @Summary Get post by slug
// @Description Returns a single published blog post by its slug.
// @Tags blog
// @Produce json
// @Param slug path string true "Post slug"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /blog/posts/slug/{slug} [get]
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

// CreatePost creates a new blog post.
// @Summary Create post
// @Description Creates a blog post. Slug defaults to a generated slug from the title. Tags are associated by name.
// @Tags blog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createPostRequest true "Post details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /blog/posts [post]
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

	var actorName string
	if userID > 0 {
		var actor models.User
		h.DB.First(&actor, userID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     userID,
		UserName:   actorName,
		Action:     "create",
		EntityType: "post",
		EntityID:   post.ID,
		Detail:     "Created post \"" + post.Title + "\"",
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizePostJSON(h, post))
	h.flushCache()
}

// UpdatePost updates an existing blog post.
// @Summary Update post
// @Description Updates the provided fields of a post. When tags is present, the tag set is replaced entirely.
// @Tags blog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Post ID"
// @Param body body updatePostRequest true "Fields to update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blog/posts/{id} [put]
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
		Title         *string  `json:"title,omitempty"`
		Slug          *string  `json:"slug,omitempty"`
		Content       *string  `json:"content,omitempty"`
		Excerpt       *string  `json:"excerpt,omitempty"`
		Status        *string  `json:"status,omitempty"`
		FeaturedImage *string  `json:"featuredImage,omitempty"`
		CategoryID    *uint    `json:"categoryId,omitempty"`
		Tags          []string `json:"tags,omitempty"`
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

	if req.Tags != nil {
		h.DB.First(&post, id)
		h.DB.Model(&post).Association("Tags").Clear()
		for _, tagName := range req.Tags {
			var tag models.Tag
			if err := h.DB.Where("name = ?", tagName).First(&tag).Error; err == nil {
				h.DB.Model(&post).Association("Tags").Append(&tag)
			}
		}
	}

	h.DB.First(&post, id)

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
		EntityType: "post",
		EntityID:   post.ID,
		Detail:     "Updated post \"" + post.Title + "\"",
	})

	json.NewEncoder(w).Encode(sanitizePostJSON(h, post))
	h.flushCache()
}

// DeletePost soft-deletes a blog post.
// @Summary Delete post
// @Description Soft-deletes a post; the record moves to trash and can be restored.
// @Tags blog
// @Produce json
// @Security BearerAuth
// @Param id path int true "Post ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blog/posts/{id} [delete]
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
		EntityType: "post",
		EntityID:   post.ID,
		Detail:     "Deleted post \"" + post.Title + "\"",
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
	h.flushCache()
}

// BatchPosts performs a bulk action on posts.
// @Summary Batch post actions
// @Description Applies an action (delete, publish, draft) to multiple posts.
// @Tags blog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body batchRequest true "Action and post IDs"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Router /blog/posts/batch [post]
func (h *Handler) BatchPosts(w http.ResponseWriter, r *http.Request) {
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
	h.flushCache()
}

// ToggleLike likes or unlikes a post for the current user.
// @Summary Like / unlike a post
// @Description Toggles the current user's like on a post and returns the new liked state and count.
// @Tags blog
// @Produce json
// @Security BearerAuth
// @Param id path int true "Post ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blog/posts/{id}/like [post]
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
	h.flushCache()
}

func sanitizeTag(t models.Tag) map[string]interface{} {
	return map[string]interface{}{
		"id":   t.ID,
		"name": t.Name,
		"slug": t.Slug,
	}
}

// GetTags returns all blog tags (public).
// @Summary List tags
// @Description Returns all blog tags.
// @Tags blog
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /blog/tags [get]
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

// CreateTag creates a blog tag (admin only).
// @Summary Create tag
// @Description Creates a blog tag. Slug defaults to a generated slug from the name.
// @Tags blog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createTagRequest true "Tag details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /blog/tags [post]
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
	h.flushCache()
}

func sanitizeCategory(c models.Category) map[string]interface{} {
	return map[string]interface{}{
		"id":          c.ID,
		"name":        c.Name,
		"slug":        c.Slug,
		"description": c.Description,
	}
}

// GetCategories returns all blog categories (public).
// @Summary List categories
// @Description Returns all blog categories.
// @Tags blog
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /blog/categories [get]
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

// CreateCategory creates a blog category (admin only).
// @Summary Create category
// @Description Creates a blog category. Slug defaults to a generated slug from the name.
// @Tags blog
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createCategoryRequest true "Category details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /blog/categories [post]
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
	h.flushCache()
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
