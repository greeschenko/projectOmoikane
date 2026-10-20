package events

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// TestRelay_ConcurrentRelaysPublishEachEventOnce is the end-to-end regression
// for the Phase 37 finding: EVERY service runs its own Relay over the ONE
// shared omoikane outbox table, so two relays polling at the same time used to
// both publish the same rows (26% of events observed on the wire twice, which
// then made the audit consumer log a duplicate-key ERROR per event).
func TestRelay_ConcurrentRelaysPublishEachEventOnce(t *testing.T) {
	db := setupOutboxDB(t)
	ctx := context.Background()

	const events = 20
	for i := 0; i < events; i++ {
		ev := sampleEvent()
		ev.ID = fmt.Sprintf("evt-relay-%d", i)
		if _, err := NewGormOutboxStore(db).Enqueue(ctx, ev); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	// Three relays sharing the table, as three services do in compose/k8s.
	var (
		mu      sync.Mutex
		publish = map[string]int{}
	)
	newRelay := func() *Relay {
		r := NewRelay(NewGormOutboxStore(db), nil, DefaultConfig())
		r.PublishOverride = func(_ context.Context, row OutboxEvent) error {
			mu.Lock()
			defer mu.Unlock()
			publish[row.Payload]++
			return nil
		}
		return r
	}
	relays := []*Relay{newRelay(), newRelay(), newRelay()}

	var wg sync.WaitGroup
	for _, r := range relays {
		wg.Add(1)
		go func(r *Relay) {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				if _, err := r.RunOnce(ctx); err != nil {
					t.Errorf("run once: %v", err)
					return
				}
			}
		}(r)
	}
	wg.Wait()

	if len(publish) != events {
		t.Fatalf("expected %d distinct events published, got %d", events, len(publish))
	}
	for payload, n := range publish {
		if n != 1 {
			var id string
			if len(payload) > 0 {
				id = payload
			}
			t.Errorf("event published %d times (want 1): %.80s", n, id)
		}
	}

	// Everything must end up sent, not stranded inflight.
	var stuck int64
	if err := db.Model(&OutboxEvent{}).
		Where("status IN ?", []string{OutboxPending, OutboxInflight}).Count(&stuck).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if stuck != 0 {
		t.Fatalf("%d rows left pending/inflight after all relays finished", stuck)
	}
}

// TestRelay_FailedPublishReleasesClaimForRetry — a failed publish must put the
// row back to pending (attempt counted) rather than leaving it claimed.
func TestRelay_FailedPublishReleasesClaimForRetry(t *testing.T) {
	db := setupOutboxDB(t)
	ctx := context.Background()
	store := NewGormOutboxStore(db)

	if _, err := store.Enqueue(ctx, sampleEvent()); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	cfg := DefaultConfig()
	cfg.OutboxMaxAttempts = 3
	relay := NewRelay(store, nil, cfg)
	relay.PublishOverride = func(context.Context, OutboxEvent) error {
		return fmt.Errorf("kafka down")
	}

	if _, err := relay.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}

	var row OutboxEvent
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != OutboxPending {
		t.Fatalf("failed publish must release the claim back to pending, got %s", row.Status)
	}
	if row.Attempts != 1 || row.LastError == "" {
		t.Fatalf("attempt not recorded: attempts=%d last_error=%q", row.Attempts, row.LastError)
	}

	// A healthy relay (nil override => real publish path is stubbed here by
	// swapping the override) must now be able to claim and send it.
	ok := NewRelay(store, nil, cfg)
	ok.PublishOverride = func(context.Context, OutboxEvent) error { return nil }
	if n, err := ok.RunOnce(ctx); err != nil || n != 1 {
		t.Fatalf("retry claim: n=%d err=%v", n, err)
	}
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != OutboxSent {
		t.Fatalf("expected sent after retry, got %s", row.Status)
	}
	if row.SentAt == nil {
		t.Fatal("sent_at not set")
	}
}

// TestRelay_RunOnceCountsOnlyFinishedRows — claimed-but-failed rows count as
// attempted; nothing pending means nothing counted.
func TestRelay_RunOnceCountsOnlyFinishedRows(t *testing.T) {
	db := setupOutboxDB(t)
	ctx := context.Background()
	store := NewGormOutboxStore(db)

	if _, err := store.Enqueue(ctx, sampleEvent()); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	relay := NewRelay(store, nil, DefaultConfig())
	relay.PublishOverride = func(context.Context, OutboxEvent) error { return nil }

	if n, err := relay.RunOnce(ctx); err != nil || n != 1 {
		t.Fatalf("first run: n=%d err=%v", n, err)
	}
	if n, err := relay.RunOnce(ctx); err != nil || n != 0 {
		t.Fatalf("empty run: n=%d err=%v (row must not be republished)", n, err)
	}

	// The relay also honours the batch size.
	for i := 0; i < 3; i++ {
		ev := sampleEvent()
		ev.ID = fmt.Sprintf("evt-batch-%d", i)
		if _, err := store.Enqueue(ctx, ev); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	cfg := DefaultConfig()
	cfg.RelayBatchSize = 2
	batched := NewRelay(store, nil, cfg)
	batched.PublishOverride = relay.PublishOverride
	if n, err := batched.RunOnce(ctx); err != nil || n != 2 {
		t.Fatalf("batch run: n=%d err=%v (want 2)", n, err)
	}
}
