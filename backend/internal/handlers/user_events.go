package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// createUserAndEmit creates a user and — when the handler is wired with an
// outbox (the auth service) — appends the user.registered CloudEvent inside
// the SAME DB transaction, so either both the user row and the outbox row
// commit together or neither does (transactional outbox).
//
// The monolith leaves Handler.Outbox nil (single-writer: only the auth service
// emits auth events), in which case the behaved path is a plain Create identical
// to the pre-Phase-29 behaviour (no transaction, no event).
func (h *Handler) createUserAndEmit(ctx context.Context, user *models.User) error {
	if h.Outbox == nil {
		return h.DB.Create(user).Error
	}
	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return h.enqueueUserRegistered(ctx, tx, *user)
	})
}

// enqueueUserRegistered marshals the user.registered payload contract
// (schemas/user.registered.json: {id, email, role}) and appends it to the
// outbox bound to tx, inheriting the caller's transaction.
func (h *Handler) enqueueUserRegistered(ctx context.Context, tx *gorm.DB, user models.User) error {
	payload, err := json.Marshal(map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	})
	if err != nil {
		return fmt.Errorf("handlers: marshal user.registered payload: %w", err)
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceAuth,
		Type:        events.TypeUserRegistered,
		Subject:     fmt.Sprintf("user/%d", user.ID),
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if _, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev); err != nil {
		return fmt.Errorf("handlers: enqueue user.registered: %w", err)
	}
	return nil
}

// emitAuthLogin appends the auth.login CloudEvent when the handler is wired
// with an outbox (the auth service — Phase 32: login is now an audit event on
// the backbone instead of the retired HTTP audit.Emit call). Login is a read
// (no business write to tie the event to), so the enqueue runs in its own
// small transaction for the same either-or-nothing guarantees as the other
// emissions.
func (h *Handler) emitAuthLogin(ctx context.Context, user models.User, method string) {
	if h.Outbox == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"id":     user.ID,
		"email":  user.Email,
		"method": method,
	})
	if err != nil {
		log.Printf("handlers: marshal auth.login payload: %v", err)
		return
	}
	ev := events.CloudEvent{
		SpecVersion: "1.0",
		ID:          events.NewEventID(),
		Source:      events.SourceAuth,
		Type:        events.TypeAuthLogin,
		Subject:     fmt.Sprintf("user/%d", user.ID),
		Time:        time.Now().UTC(),
		Data:        payload,
	}
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		_, err := events.NewGormOutboxStore(tx).Enqueue(ctx, ev)
		return err
	}); err != nil {
		// Login must not fail because audit logging failed — log and proceed.
		log.Printf("handlers: enqueue auth.login: %v", err)
	}
}
