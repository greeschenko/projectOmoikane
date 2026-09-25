package models

import (
	"time"

	"gorm.io/gorm"
)

// WebhookSubscription delivers select platform events to an external HTTP
// endpoint (Phase 35). One subscription = one event type -> one URL. The
// optional Secret key signs each delivery with HMAC-SHA256 so receivers can
// authenticate the request (X-Omoikane-Signature header).
//
// Unlike ApiToken, the secret MUST stay recoverable at rest: it is the HMAC
// key used to sign outbound payloads, so a hash would be useless to the
// signer. This is a documented trade-off (encryption at rest is a follow-up).
// The Secret field never serializes (json:"-"); the raw value is only shown
// once at creation.
type WebhookSubscription struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
	// EventType is the full CloudEvents type this subscription receives (the
	// 7 live backbone types, e.g. org.omoikane.content.post.published.v1).
	EventType string `gorm:"not null;index" json:"eventType"`
	// URL receives the CloudEvents JSON POST (delivery retries with backoff).
	URL string `gorm:"not null" json:"url"`
	// Secret (optional) is the HMAC-SHA256 key; empty disables signing.
	Secret string `gorm:"type:text" json:"-"`
	// Active toggles membership without deleting the subscription. There is no
	// `default:true` GORM tag on purpose: GORM replaces zero values with the
	// default tag during INSERT, which would silently force every subscription
	// to active=true — and inactive IS a legitimate explicit value here. The
	// default (true) is enforced in CreateWebhook instead.
	Active bool `json:"active"`
}

// WebhookDelivery is one delivery attempt ledger row (the delivery-log UI).
//
// Status lifecycle: pending -> delivered | failed -> (failed retries) -> expired
// The terminal "expired" state is the webhook module's DLQ equivalent — the
// endpoint exhausted deliveryMaxAttempts and the row stays visible so the
// operator can see exactly how things degraded.
//
// The unique (subscription_id, event_id) index makes the Kafka consumer
// idempotent under at-least-once redelivery: a redelivered event skips rows
// that already exist instead of duplicating deliveries.
type WebhookDelivery struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deletedAt"`
	SubscriptionID uint           `gorm:"index;not null;uniqueIndex:idx_wh_delivery_sub_event" json:"subscriptionId"`
	// EventID is the CloudEvent id that produced this row ("test" rows carry a
	// synthetic ping event id).
	EventID   string `gorm:"not null;uniqueIndex:idx_wh_delivery_sub_event" json:"eventId"`
	EventType string `gorm:"not null;index" json:"eventType"`
	// Payload is the exact CloudEvents JSON delivered (or attempted).
	Payload string `gorm:"type:text" json:"payload"`
	// Attempts counts HTTP delivery attempts so far (0 = not yet delivered).
	Attempts int `gorm:"default:0" json:"attempts"`
	// HTTPStatus is the last response status (0 = transport-level failure).
	HTTPStatus int    `json:"httpStatus"`
	Status     string `gorm:"default:pending;not null;index" json:"status"`
	// Error is the last failure detail ("" when not yet failed).
	Error string `gorm:"type:text" json:"error"`
	// NextAttemptAt schedules the next retry (zero/omitted once terminal).
	NextAttemptAt time.Time `gorm:"index" json:"nextAttemptAt"`
}