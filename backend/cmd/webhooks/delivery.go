package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// Delivery statuses (see models.WebhookDelivery).
const (
	statusPending   = "pending"
	statusDelivered = "delivered"
	statusFailed    = "failed"
	statusExpired   = "expired"
)

// deliveryWorker is the webhook delivery pump (Phase 35). It polls due
// `pending` rows and POSTs each payload to the subscription URL, signing with
// HMAC-SHA256 when the subscription has a secret. Failures retry with
// exponential backoff; after deliveryMaxAttempts a row goes terminal (expired)
// — the module's DLQ-equivalent state, visible in the delivery-log UI.
//
// The pump runs OUTSIDE the Kafka consumer loop on purpose: the consumer only
// enqueues pending rows (fast ACK, idempotent via the unique index), and the
// HTTP fan-out + retries live here. Consumer-level retries therefore can never
// duplicate deliveries.
type deliveryWorker struct {
	db *gorm.DB
	// client performs the HTTP POSTs (timeout bounded; tests inject a client
	// pointed at an httptest sink).
	client *http.Client

	pollInterval time.Duration
	batchSize    int
	// maxAttempts is the total number of HTTP attempts before a row expires
	// (1 initial + maxAttempts-1 retries).
	maxAttempts int
	// baseBackoff is the first retry delay; each further retry doubles it,
	// capped at backoffCap.
	baseBackoff time.Duration
	backoffCap  time.Duration
}

// newDeliveryWorker constructs the pump with production defaults.
func newDeliveryWorker(db *gorm.DB) *deliveryWorker {
	return &deliveryWorker{
		db:           db,
		client:       &http.Client{Timeout: 10 * time.Second},
		pollInterval: 500 * time.Millisecond,
		batchSize:    50,
		maxAttempts:  6,
		baseBackoff:  time.Second,
		backoffCap:   60 * time.Second,
	}
}

// Run polls due deliveries until ctx is cancelled (mirrors events.Relay).
func (w *deliveryWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("webhooks-service: delivery pump stopped")
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				log.Printf("WARNING: webhooks-service: delivery pump poll error: %v", err)
			}
		}
	}
}

// RunOnce delivers one batch of due pending/failed rows. Returns the number of
// rows attempted. Exposed for tests (deterministic pump steps without sleeping).
func (w *deliveryWorker) RunOnce(ctx context.Context) (int, error) {
	var due []models.WebhookDelivery
	// pending = awaiting first delivery; failed = a past attempt failed and a
	// retry is scheduled (next_attempt_at in the past means it is due NOW).
	if err := w.db.Where("status IN ? AND next_attempt_at <= ?", []string{statusPending, statusFailed}, time.Now().UTC()).
		Order("id asc").Limit(w.batchSize).Find(&due).Error; err != nil {
		return 0, err
	}
	for i := range due {
		w.deliver(ctx, &due[i])
	}
	return len(due), nil
}

// deliver performs one HTTP delivery for the due row. Always updates the row
// (delivered, or failed/expired + retry scheduling) even on transport errors —
// a redeliver of the same row is impossible because RunOnce only selects
// pending/failed rows.
func (w *deliveryWorker) deliver(ctx context.Context, d *models.WebhookDelivery) {
	var sub models.WebhookSubscription
	if err := w.db.First(&sub, d.SubscriptionID).Error; err != nil {
		// Subscription deleted: nothing left to deliver to. Terminate quietly
		// instead of retrying forever against a missing target.
		w.updateRow(d, map[string]interface{}{
			"status":          statusExpired,
			"error":           "subscription not found",
			"next_attempt_at": nil,
		})
		return
	}

	body := []byte(d.Payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		w.recordFailure(d, fmt.Errorf("build request: %w", err))
		return
	}
	req.Header.Set("Content-Type", "application/cloudevents+json")
	req.Header.Set("User-Agent", "omoikane-webhooks/1.0")
	if sub.Secret != "" {
		req.Header.Set("X-Omoikane-Signature", signPayload(sub.Secret, body))
	}

	resp, err := w.client.Do(req)
	if err != nil {
		w.recordFailure(d, fmt.Errorf("request failed: %w", err))
		return
	}
	defer resp.Body.Close()
	// Drain so the connection can be reused; cap the read to bound memory.
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.updateRow(d, map[string]interface{}{
			"status":          statusDelivered,
			"attempts":        w.nextAttempts(d),
			"http_status":     resp.StatusCode,
			"error":           "",
			"next_attempt_at": nil,
		})
		return
	}
	w.recordFailure(d, fmt.Errorf("HTTP %d", resp.StatusCode))
}

// recordFailure increments the attempt counter and either schedules the next
// retry (exponential backoff) or expires the row after maxAttempts.
func (w *deliveryWorker) recordFailure(d *models.WebhookDelivery, cause error) {
	attempts := w.nextAttempts(d)
	if attempts >= w.maxAttempts {
		w.updateRow(d, map[string]interface{}{
			"attempts":        attempts,
			"status":          statusExpired,
			"http_status":     d.HTTPStatus,
			"error":           cause.Error(),
			"next_attempt_at": nil,
		})
		return
	}
	w.updateRow(d, map[string]interface{}{
		"attempts":        attempts,
		"status":          statusFailed,
		"http_status":     d.HTTPStatus,
		"error":           cause.Error(),
		"next_attempt_at": time.Now().Add(w.backoff(attempts)),
	})
}

// nextAttempts derives the attempt number from the row (Attempts is the
// number of past deliveries; the current one bumps it by one).
func (w *deliveryWorker) nextAttempts(d *models.WebhookDelivery) int {
	return d.Attempts + 1
}

// backoff returns the delay before the next retry after the current attempt
// number: base, 2*base, 4*base ... capped at backoffCap.
func (w *deliveryWorker) backoff(attempt int) time.Duration {
	delay := w.baseBackoff
	for i := 1; i < attempt && delay < w.backoffCap; i++ {
		delay *= 2
	}
	if delay > w.backoffCap {
		delay = w.backoffCap
	}
	return delay
}

// updateRow writes a status transition without touching other fields.
func (w *deliveryWorker) updateRow(d *models.WebhookDelivery, fields map[string]interface{}) {
	fields["updated_at"] = time.Now().UTC()
	if err := w.db.Model(d).Updates(fields).Error; err != nil {
		log.Printf("WARNING: webhooks-service: update delivery %d: %v", d.ID, err)
	}
}

// signPayload returns the X-Omoikane-Signature value: `sha256=<hex hmac>` over
// the exact request body, keyed by the subscription secret.
func signPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
