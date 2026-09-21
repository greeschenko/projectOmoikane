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

// createMediaAndEmit creates a media item and — when the handler is wired with
// an outbox (the media service) — appends the media.uploaded CloudEvent inside
// the SAME DB transaction.
//
// The monolith leaves Handler.Outbox nil (single-writer: only the media service
// emits media events), in which case the behaved path is a plain Create.
func (h *Handler) createMediaAndEmit(ctx context.Context, item *models.MediaItem) error {
	if h.Outbox == nil {
		return h.DB.Create(item).Error
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		return h.enqueueMediaUploaded(ctx, tx, *item)
	})
}

// enqueueMediaUploaded marshals the media.uploaded payload contract
// (schemas/media.uploaded.json: {id, filename, alt, url, thumbUrl, size,
// uploadedAt}) and appends it to the outbox bound to tx. URL/thumbUrl follow
// the same MEDIA_BASE_URL rules as the API responses.
func (h *Handler) enqueueMediaUploaded(ctx context.Context, tx *gorm.DB, item models.MediaItem) error {
	uploadedAt := item.CreatedAt
	if uploadedAt.IsZero() {
		uploadedAt = time.Now()
	}
	payload, err := json.Marshal(map[string]any{
		"id":         item.ID,
		"filename":   item.Filename,
		"alt":        item.Alt,
		"url":        h.mediaURL(item),
		"thumbUrl":   h.thumbURL(item),
		"size":       item.Size,
		"uploadedAt": uploadedAt,
	})
	if err != nil {
		return fmt.Errorf("handlers: marshal media.uploaded payload: %w", err)
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceMedia,
		Type:        events.TypeMediaUploaded,
		Subject:     fmt.Sprintf("media/%d", item.ID),
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if _, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev); err != nil {
		return fmt.Errorf("handlers: enqueue media.uploaded: %w", err)
	}
	return nil
}
