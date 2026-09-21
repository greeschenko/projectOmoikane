package events

import (
	"context"
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
	if row.Status != OutboxPending {
		t.Fatalf("expected status pending, got %s", row.Status)
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
