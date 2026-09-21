package events

import (
	"context"
	"log"
	"time"
)

// Relay is the outbox publisher worker. It polls pending outbox rows on an
// interval and publishes each via a Producer, then marks rows sent (or
// failed after OutboxMaxAttempts). Runs in every service writing domain events.
//
// Design notes:
//   - at-least-once: a crash between publish and MarkSent redelivers the same
//     event; downstream consumers must be idempotent (see Handler docs).
//   - batch: RelayBatchSize rows per poll keep memory bounded and backpressure
//     naturally pushes retries to the next interval when Kafka is slow/down.
type Relay struct {
	store    OutboxStore
	producer Producer
	cfg      Config

	// PublishOverride lets tests stub the publish step (nil => real producer).
	PublishOverride func(context.Context, OutboxEvent) error
}

// NewRelay wires a Relay with the given outbox store, producer, and config.
func NewRelay(store OutboxStore, producer Producer, cfg Config) *Relay {
	return &Relay{store: store, producer: producer, cfg: cfg}
}

// RunOnce publishes one batch of pending outbox events. Returns the number of
// events attempted (successful publishes + final failures; rows left pending
// for retry are not counted).
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	limit := r.cfg.RelayBatchSize
	if limit <= 0 {
		limit = 100
	}
	pending, err := r.store.Pending(ctx, limit)
	if err != nil {
		return 0, err
	}
	attempted := 0
	for _, row := range pending {
		attempted++
		if err := r.publishRow(ctx, row); err != nil {
			log.Printf("events: relay: %v", err)
		}
	}
	return attempted, nil
}

// publishRow publishes one outbox row.
func (r *Relay) publishRow(ctx context.Context, row OutboxEvent) error {
	if r.PublishOverride != nil {
		return r.PublishOverride(ctx, row)
	}
	ev, err := UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		// Corrupt payload: nothing to publish — mark failed so it stops
		// blocking the queue and gets operator attention.
		return r.store.MarkAttempt(ctx, row.ID, err, r.cfg.OutboxMaxAttempts)
	}
	if err := r.producer.Publish(ctx, ev); err != nil {
		return r.store.MarkAttempt(ctx, row.ID, err, r.cfg.OutboxMaxAttempts)
	}
	return r.store.MarkSent(ctx, row.ID)
}

// Run blocks, polling the outbox every RelayInterval until ctx is cancelled.
// Persistent publish failures do not stop the loop: rows stay pending and are
// retried (up to OutboxMaxAttempts before being marked failed), so a Kafka
// outage pauses delivery without losing events.
func (r *Relay) Run(ctx context.Context) {
	interval := r.cfg.RelayInterval
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("events: relay stopped")
			return
		case <-ticker.C:
			if _, err := r.RunOnce(ctx); err != nil {
				log.Printf("events: relay poll error: %v", err)
			}
		}
	}
}
