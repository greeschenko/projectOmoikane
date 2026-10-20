package events

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"encoding/json"
)

func eventsTestDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
}

// setupOutboxDB opens a GORM connection, migrates ONLY the outbox table, and
// wipes stale rows so tests run on a clean slate without touching other tables.
func setupOutboxDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(eventsTestDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		t.Fatalf("events: connect test DB: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(3)
	}
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			sqlDB.Close()
		}
	})
	if err := db.Exec("DROP TABLE IF EXISTS outbox_events").Error; err != nil {
		t.Fatalf("events: drop outbox: %v", err)
	}
	if err := MigrateOutbox(db); err != nil {
		t.Fatalf("events: migrate outbox: %v", err)
	}
	return db
}

func sampleEvent() CloudEvent {
	return CloudEvent{
		SpecVersion: "1.0",
		ID:          "evt-1",
		Source:      SourceContent,
		Type:        TypePostPublished,
		Subject:     "post/42",
		Time:        time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC),
		Data:        json.RawMessage(`{"id":42,"status":"published"}`),
	}
}

func TestOutboxStore_EnqueuePendingMarkSent(t *testing.T) {
	db := setupOutboxDB(t)
	store := NewGormOutboxStore(db)
	ctx := context.Background()

	ev := sampleEvent()
	id, err := store.Enqueue(ctx, ev)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	rows, err := store.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 pending row, got %d", len(rows))
	}
	row := rows[0]
	if row.ID != id || row.EventType != ev.Type || row.Subject != ev.Subject {
		t.Fatalf("row mismatch: %+v", row)
	}
	// Pending CLAIMS what it hands out (pending -> inflight) so a second relay
	// polling the same table cannot publish the same row again.
	if row.Status != OutboxInflight {
		t.Fatalf("expected status inflight (claimed), got %s", row.Status)
	}
	// Payload must round-trip back to the original envelope.
	parsed, err := UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		t.Fatalf("payload unmarshal: %v", err)
	}
	if parsed.ID != ev.ID || parsed.Type != ev.Type {
		t.Fatalf("payload mismatch: %+v", parsed)
	}

	if err := store.MarkSent(ctx, id); err != nil {
		t.Fatalf("mark sent: %v", err)
	}
	rows, _ = store.Pending(ctx, 10)
	if len(rows) != 0 {
		t.Fatalf("expected no pending rows after mark sent, got %d", len(rows))
	}
}

// TestOutboxStore_PendingClaimsRows — the Phase 37 regression. Five services
// run their own Relay over the ONE shared omoikane outbox table; without the
// claim every relay published every row (measured: 26% of events on the wire
// twice). A claimed row must be invisible to the next poll, whichever store
// instance (service) asks.
func TestOutboxStore_PendingClaimsRows(t *testing.T) {
	db := setupOutboxDB(t)
	serviceA := NewGormOutboxStore(db) // "auth-service"
	serviceB := NewGormOutboxStore(db) // "content-service"
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		ev := sampleEvent()
		ev.ID = fmt.Sprintf("evt-%d", i)
		if _, err := serviceA.Enqueue(ctx, ev); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	first, err := serviceA.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending A: %v", err)
	}
	if len(first) != 3 {
		t.Fatalf("service A expected 3 rows, got %d", len(first))
	}

	// The other service's relay must see nothing: the rows are already claimed.
	second, err := serviceB.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending B: %v", err)
	}
	if len(second) != 0 {
		ids := make([]uint, len(second))
		for i, r := range second {
			ids[i] = r.ID
		}
		t.Fatalf("service B re-claimed rows already held by service A: %v", ids)
	}

	// Once A publishes and marks them sent they stay out of B's way for good.
	for _, row := range first {
		if err := serviceA.MarkSent(ctx, row.ID); err != nil {
			t.Fatalf("mark sent: %v", err)
		}
	}
	third, err := serviceB.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending B after sent: %v", err)
	}
	if len(third) != 0 {
		t.Fatalf("expected 0 rows after A published, service B got %d", len(third))
	}
}

// TestOutboxStore_ReclaimsStaleInflightRows — crash recovery. A relay that
// dies mid-batch leaves its claim behind; the reclaim puts the row back so the
// events are not stranded forever (at-least-once: consumers tolerate the
// duplicate).
func TestOutboxStore_ReclaimsStaleInflightRows(t *testing.T) {
	db := setupOutboxDB(t)
	store := NewGormOutboxStore(db)
	ctx := context.Background()

	id, err := store.Enqueue(ctx, sampleEvent())
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	claimed, err := store.Pending(ctx, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v (rows=%d)", err, len(claimed))
	}

	// Fresh claim: not reclaimed yet.
	rows, err := store.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending (fresh): %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("freshly claimed row must not be reclaimed, got %d", len(rows))
	}

	// Age the claim past the timeout, as if the publishing relay had died.
	stale := time.Now().Add(-OutboxInflightTimeout - time.Minute)
	if err := db.Model(&OutboxEvent{}).Where("id = ?", id).
		Updates(map[string]any{"updated_at": stale}).Error; err != nil {
		t.Fatalf("age claim: %v", err)
	}

	rows, err = store.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending (stale): %v", err)
	}
	if len(rows) != 1 || rows[0].ID != id {
		t.Fatalf("stale claim was not reclaimed, got %d rows", len(rows))
	}
}

// TestOutboxStore_PendingRespectsLimit — the claim must respect the batch size
// and leave the rest pending for the next poll.
func TestOutboxStore_PendingRespectsLimit(t *testing.T) {
	db := setupOutboxDB(t)
	store := NewGormOutboxStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		ev := sampleEvent()
		ev.ID = fmt.Sprintf("evt-limit-%d", i)
		if _, err := store.Enqueue(ctx, ev); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	rows, err := store.Pending(ctx, 2)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (limit), got %d", len(rows))
	}
	// Oldest first.
	if rows[0].ID > rows[1].ID {
		t.Fatalf("expected oldest-first order, got %d then %d", rows[0].ID, rows[1].ID)
	}
	// The remaining 3 stay claimable (they are still pending).
	for _, row := range rows {
		if err := store.MarkSent(ctx, row.ID); err != nil {
			t.Fatalf("mark sent: %v", err)
		}
	}
	rest, err := store.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending rest: %v", err)
	}
	if len(rest) != 3 {
		t.Fatalf("expected 3 remaining rows, got %d", len(rest))
	}
}

func TestOutboxStore_MarkAttemptExhaustsToFailed(t *testing.T) {
	db := setupOutboxDB(t)
	store := NewGormOutboxStore(db)
	ctx := context.Background()

	id, err := store.Enqueue(ctx, sampleEvent())
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// maxAttempts=3 => attempts 1,2 stay pending; 3 becomes failed.
	for want := 1; want <= 3; want++ {
		if err := store.MarkAttempt(ctx, id, context.DeadlineExceeded, 3); err != nil {
			t.Fatalf("mark attempt %d: %v", want, err)
		}
		var row OutboxEvent
		if err := db.First(&row, id).Error; err != nil {
			t.Fatalf("reload: %v", err)
		}
		if row.Attempts != want {
			t.Fatalf("attempt %d: attempts=%d", want, row.Attempts)
		}
		wantStatus := OutboxPending
		if want >= 3 {
			wantStatus = OutboxFailed
		}
		if row.Status != wantStatus {
			t.Fatalf("attempt %d: status=%s want=%s", want, row.Status, wantStatus)
		}
		if row.LastError == "" {
			t.Fatalf("attempt %d: last_error not recorded", want)
		}
	}
}

func TestOutboxStore_EnqueueMarshalRoundTrip(t *testing.T) {
	db := setupOutboxDB(t)
	store := NewGormOutboxStore(db)
	ctx := context.Background()

	ev := sampleEvent()
	id, err := store.Enqueue(ctx, ev)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	var row OutboxEvent
	if err := db.First(&row, id).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	sent, err := UnmarshalCloudEvent([]byte(row.Payload))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(sent.Data) != string(ev.Data) {
		t.Fatalf("data mismatch:\n got: %s\nwant: %s", sent.Data, ev.Data)
	}
}
