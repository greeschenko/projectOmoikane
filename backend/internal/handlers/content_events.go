package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// createPageAndEmit creates a page and — when the handler is wired with an
// outbox (the content service) — appends the page.published CloudEvent inside
// the SAME DB transaction whenever the page is created as published.
//
// The monolith leaves Handler.Outbox nil (single-writer: only the content
// service emits content events), in which case the behaved path is a plain
// Create identical to the pre-Phase-30 behaviour.
func (h *Handler) createPageAndEmit(ctx context.Context, page *models.Page) error {
	if h.Outbox == nil {
		return h.DB.Create(page).Error
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(page).Error; err != nil {
			return err
		}
		if page.Status == "published" {
			return h.enqueuePagePublished(ctx, tx, *page)
		}
		return nil
	})
}

// updatePageAndEmit applies the given column updates to a page and re-reads it.
// When the outbox is wired and the update transitions the page TO published, a
// page.published CloudEvent is enqueued inside the same transaction.
func (h *Handler) updatePageAndEmit(ctx context.Context, prev models.Page, updates map[string]any) (models.Page, error) {
	id := prev.ID
	if h.Outbox == nil {
		if len(updates) > 0 {
			if err := h.DB.Model(&models.Page{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return prev, err
			}
		}
		var page models.Page
		if err := h.DB.First(&page, id).Error; err != nil {
			return prev, err
		}
		return page, nil
	}

	var page models.Page
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.Page{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := tx.First(&page, id).Error; err != nil {
			return err
		}
		if page.Status == "published" && prev.Status != "published" {
			return h.enqueuePagePublished(ctx, tx, page)
		}
		return nil
	})
	return page, err
}

// publishPagesAndEmit marks the given pages published. When the outbox is wired,
// one page.published event is enqueued per page that actually transitions to
// published (previously draft).
func (h *Handler) publishPagesAndEmit(ctx context.Context, ids []uint) error {
	if h.Outbox == nil {
		return h.DB.Model(&models.Page{}).Where("id IN ?", ids).Update("status", "published").Error
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		var pages []models.Page
		if err := tx.Where("id IN ?", ids).Find(&pages).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Page{}).Where("id IN ?", ids).Update("status", "published").Error; err != nil {
			return err
		}
		for _, p := range pages {
			if p.Status != "published" {
				if err := h.enqueuePagePublished(ctx, tx, p); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// enqueuePagePublished marshals the page.published payload contract
// (schemas/page.published.json: {id, slug, title, status, publishedAt}) and
// appends it to the outbox bound to tx.
func (h *Handler) enqueuePagePublished(ctx context.Context, tx *gorm.DB, page models.Page) error {
	payload, err := json.Marshal(map[string]any{
		"id":          page.ID,
		"slug":        page.Slug,
		"title":       page.Title,
		"status":      "published",
		"publishedAt": time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("handlers: marshal page.published payload: %w", err)
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceContent,
		Type:        events.TypePagePublished,
		Subject:     fmt.Sprintf("page/%d", page.ID),
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if _, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev); err != nil {
		return fmt.Errorf("handlers: enqueue page.published: %w", err)
	}
	return nil
}

// createPostAndEmit creates a blog post and associates its tags. When the
// outbox is wired, creation + tag association + post.published enqueue run in
// one transaction (created-as-published posts emit).
func (h *Handler) createPostAndEmit(ctx context.Context, post *models.BlogPost, tagNames []string) error {
	if h.Outbox == nil {
		if err := h.DB.Create(post).Error; err != nil {
			return err
		}
		return h.associatePostTags(h.DB, post, tagNames)
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		if err := h.associatePostTags(tx, post, tagNames); err != nil {
			return err
		}
		if post.Status == "published" {
			return h.enqueuePostPublished(ctx, tx, *post)
		}
		return nil
	})
}

// updatePostAndEmit applies updates to a post, optionally replacing its tag set,
// and re-reads it. When the outbox is wired and the update transitions the post
// TO published, a post.published CloudEvent is enqueued inside the transaction.
func (h *Handler) updatePostAndEmit(ctx context.Context, prev models.BlogPost, updates map[string]any, tagNames []string) (models.BlogPost, error) {
	id := prev.ID
	replaceTags := tagNames != nil
	if h.Outbox == nil {
		if len(updates) > 0 {
			if err := h.DB.Model(&models.BlogPost{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return prev, err
			}
		}
		var post models.BlogPost
		if err := h.DB.First(&post, id).Error; err != nil {
			return prev, err
		}
		if replaceTags {
			if err := h.DB.Model(&post).Association("Tags").Clear(); err != nil {
				return prev, err
			}
			if err := h.associatePostTags(h.DB, &post, tagNames); err != nil {
				return prev, err
			}
		}
		if err := h.DB.First(&post, id).Error; err != nil {
			return prev, err
		}
		return post, nil
	}

	var post models.BlogPost
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.BlogPost{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if replaceTags {
			if err := tx.Model(&models.BlogPost{}).Where("id = ?", id).First(&post).Error; err != nil {
				return err
			}
			if err := tx.Model(&post).Association("Tags").Clear(); err != nil {
				return err
			}
			if err := h.associatePostTags(tx, &post, tagNames); err != nil {
				return err
			}
		}
		if err := tx.First(&post, id).Error; err != nil {
			return err
		}
		if post.Status == "published" && prev.Status != "published" {
			return h.enqueuePostPublished(ctx, tx, post)
		}
		return nil
	})
	return post, err
}

// publishPostsAndEmit marks the given posts published. When the outbox is wired,
// one post.published event is enqueued per post that transitions to published.
func (h *Handler) publishPostsAndEmit(ctx context.Context, ids []uint) error {
	if h.Outbox == nil {
		return h.DB.Model(&models.BlogPost{}).Where("id IN ?", ids).Update("status", "published").Error
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		var posts []models.BlogPost
		if err := tx.Where("id IN ?", ids).Find(&posts).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.BlogPost{}).Where("id IN ?", ids).Update("status", "published").Error; err != nil {
			return err
		}
		for _, p := range posts {
			if p.Status != "published" {
				if err := h.enqueuePostPublished(ctx, tx, p); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// associatePostTags links a post to every tag whose name matches an entry in
// tagNames. Unknown tag names are skipped (same behaviour as the monolith).
func (h *Handler) associatePostTags(db *gorm.DB, post *models.BlogPost, tagNames []string) error {
	for _, tagName := range tagNames {
		var tag models.Tag
		if err := db.Where("name = ?", tagName).First(&tag).Error; err == nil {
			if err := db.Model(post).Association("Tags").Append(&tag); err != nil {
				return err
			}
		}
	}
	return nil
}

// enqueuePostPublished marshals the post.published payload contract
// (schemas/post.published.json: {id, slug, title, status, categoryId, tagIds,
// publishedAt}) and appends it to the outbox bound to tx.
func (h *Handler) enqueuePostPublished(ctx context.Context, tx *gorm.DB, post models.BlogPost) error {
	var tagIds []uint
	if err := tx.Model(&post).Association("Tags").Find(&post.Tags); err == nil {
		for _, t := range post.Tags {
			tagIds = append(tagIds, t.ID)
		}
	}
	var categoryID any
	if post.CategoryID != nil {
		categoryID = *post.CategoryID
	} else {
		categoryID = nil
	}
	payload, err := json.Marshal(map[string]any{
		"id":          post.ID,
		"slug":        post.Slug,
		"title":       post.Title,
		"status":      "published",
		"categoryId":  categoryID,
		"tagIds":      tagIds,
		"publishedAt": time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("handlers: marshal post.published payload: %w", err)
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceContent,
		Type:        events.TypePostPublished,
		Subject:     fmt.Sprintf("post/%d", post.ID),
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if _, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev); err != nil {
		return fmt.Errorf("handlers: enqueue post.published: %w", err)
	}
	return nil
}
