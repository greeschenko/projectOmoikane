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

// createContactAndEmit stores a contact form submission and — when the handler
// is wired with an outbox (the messages service — Phase 32: contact form
// submissions are now audit events on the backbone) — appends the
// contact.received CloudEvent inside the SAME DB transaction.
//
// Handlers running without an outbox (tests, direct wiring) take the plain
// Create path: no event, no transaction.
func (h *Handler) createContactAndEmit(ctx context.Context, msg *models.ContactMessage) error {
	if h.Outbox == nil {
		return h.DB.Create(msg).Error
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return h.enqueueContactReceived(ctx, tx, *msg)
	})
}

// enqueueContactReceived marshals the contact.received payload contract
// (schemas/contact.received.json: {id, name, email, subject, message,
// receivedAt}) and appends it to the outbox bound to tx, inheriting the
// caller's transaction.
func (h *Handler) enqueueContactReceived(ctx context.Context, tx *gorm.DB, msg models.ContactMessage) error {
	receivedAt := msg.CreatedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now()
	}
	payload, err := json.Marshal(map[string]any{
		"id":         msg.ID,
		"name":       msg.Name,
		"email":      msg.Email,
		"subject":    msg.Subject,
		"message":    msg.Message,
		"receivedAt": receivedAt,
	})
	if err != nil {
		return fmt.Errorf("handlers: marshal contact.received payload: %w", err)
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceMessages,
		Type:        events.TypeContactReceived,
		Subject:     fmt.Sprintf("contact/%d", msg.ID),
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if _, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev); err != nil {
		return fmt.Errorf("handlers: enqueue contact.received: %w", err)
	}
	return nil
}
