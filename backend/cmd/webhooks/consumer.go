package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// webhookEventHandler is the webhooks service's Kafka consumer handler
// (Phase 35). It maps backbone CloudEvents to pending WebhookDelivery rows for
// every ACTIVE subscription matching the event type; the delivery pump does
// the actual HTTP fan-out + retries.
//
// Delivery is at-least-once end to end, so Handle must be idempotent: the
// unique (subscription_id, event_id) index already guarantees one row per
// event+subscription, and a redelivered event that finds existing rows is
// treated as success (the consumer commits instead of DLQ-ing a duplicate).
type webhookEventHandler struct {
	db *gorm.DB
}

// Handle enqueues deliveries for one CloudEvent.
func (h *webhookEventHandler) Handle(ctx context.Context, ev events.CloudEvent) error {
	var subs []models.WebhookSubscription
	if err := h.db.Where("event_type = ? AND active = ?", ev.Type, true).Find(&subs).Error; err != nil {
		return fmt.Errorf("webhooks: load subscriptions for %s: %w", ev.Type, err)
	}
	if len(subs) == 0 {
		return nil // no subscribers: ACK fast, nothing to record
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("webhooks: marshal %s event %s: %w", ev.Type, ev.ID, err)
	}
	now := time.Now().UTC()

	for _, sub := range subs {
		row := models.WebhookDelivery{
			SubscriptionID: sub.ID,
			EventID:        ev.ID,
			EventType:      ev.Type,
			Payload:        string(payload),
			Status:         statusPending,
			NextAttemptAt:  now,
		}
		if err := h.db.Create(&row).Error; err != nil {
			if h.alreadyEnqueued(sub.ID, ev.ID) {
				continue // at-least-once redelivery: row exists, treat as done
			}
			return fmt.Errorf("webhooks: enqueue delivery for %s event %s (sub %d): %w", ev.Type, ev.ID, sub.ID, err)
		}
	}
	return nil
}

// alreadyEnqueued reports whether a delivery row already exists for this
// event+subscription (behind the unique index; used to distinguish a duplicate
// from a real storage error).
func (h *webhookEventHandler) alreadyEnqueued(subscriptionID uint, eventID string) bool {
	var n int64
	h.db.Model(&models.WebhookDelivery{}).
		Where("subscription_id = ? AND event_id = ?", subscriptionID, eventID).
		Count(&n)
	return n > 0
}
