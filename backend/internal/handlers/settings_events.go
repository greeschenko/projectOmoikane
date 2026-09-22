package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// emitSettingsUpdated appends the settings.updated CloudEvent when the handler
// is wired with an outbox (the settings service — Phase 32: settings writes
// are now audit events on the backbone). The settings row (id=1) is created
// on demand and the update commits before this is called, so the enqueue runs
// in its own small transaction (an audit log must never roll back a settings
// change); a failure is logged and swallowed.
func (h *Handler) emitSettingsUpdated(ctx context.Context, settings models.SiteSetting) {
	if h.Outbox == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"siteName":    settings.SiteName,
		"tagline":     settings.Tagline,
		"blogEnabled": settings.BlogEnabled,
	})
	if err != nil {
		log.Printf("handlers: marshal settings.updated payload: %v", err)
		return
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceSettings,
		Type:        events.TypeSettingsUpdated,
		Subject:     "settings/1",
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		_, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev)
		return err
	}); err != nil {
		log.Printf("handlers: enqueue settings.updated: %v", err)
	}
}
