package events

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OutboxStatus values.
const (
	OutboxPending  = "pending"  // awaiting relay publish
	OutboxInflight = "inflight" // claimed by a relay; publish in progress
	OutboxSent     = "sent"     // relay published to Kafka
	OutboxFailed   = "failed"   // exhausted OutboxMaxAttempts; operator attention
)

// OutboxInflightTimeout is how long a claimed row may stay claimed before
// another relay takes it back. A relay marks rows inflight while it publishes;
// if its process dies mid-batch the rows would otherwise be stranded (events
// silently lost). The window only has to outlast a single publish by a wide
// margin (Kafka BatchTimeout 50ms + ack), so a crashed relay's batch is
// recovered within this bound. Recovery is at-least-once: a row published just
// before the crash goes out twice, which consumers tolerate by design.
const OutboxInflightTimeout = 2 * time.Minute

// OutboxEvent is the transactional-outbox row. Services append here IN THE
// SAME DB TRANSACTION as the business write (GORM tx), giving atomicity:
// either the business change and its event both commit, or neither does.
// A relay worker CLAIMS pending rows (pending -> inflight), publishes them to
// Kafka, then marks them sent (or failed after OutboxMaxAttempts). Claiming is
// what makes one shared outbox safe: several services relay the same table, so
// without it every relay publishes every row (Phase 37 measured 26% of events
// on the wire twice). Consumers must tolerate at-least-once semantics.
type OutboxEvent struct {
	ID        uint   `gorm:"primarykey"`
	EventType string `gorm:"size:160;not null;index"`
	Source    string `gorm:"size:80;not null"`
	Subject   string `gorm:"size:200"`
	Payload   string `gorm:"type:text;not null"` // serialized CloudEvent envelope
	Status    string `gorm:"size:16;not null;index:idx_outbox_status"`
	Attempts  int    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	SentAt    *time.Time
	LastError string `gorm:"type:text"`
}

// TableName keeps the outbox out of the default pluralization pool.
func (OutboxEvent) TableName() string { return "outbox_events" }

// OutboxStore abstracts the outbox persistence so services can inject either a
// GORM-backed store or fakes in tests.
type OutboxStore interface {
	// Enqueue appends an event to the outbox (call inside the business tx for
	// atomicity). Returns the row ID.
	Enqueue(ctx context.Context, ev CloudEvent) (uint, error)

	// Pending returns up to limit rows still to be published (oldest first),
	// CLAIMING them so that no other relay can publish the same row. Claimed
	// rows move to the inflight status until MarkSent/MarkAttempt releases
	// them; rows stranded by a crashed relay are reclaimed after
	// OutboxInflightTimeout.
	Pending(ctx context.Context, limit int) ([]OutboxEvent, error)

	// MarkSent records successful Kafka publish.
	MarkSent(ctx context.Context, id uint) error

	// MarkAttempt increments Attempts and records err; if attempts reach the
	// cap the row is marked failed (logged, not dropped).
	MarkAttempt(ctx context.Context, id uint, err error, maxAttempts int) error
}

// GormOutboxStore is the GORM-backed OutboxStore.
type GormOutboxStore struct {
	DB *gorm.DB
}

var _ OutboxStore = (*GormOutboxStore)(nil)

// NewGormOutboxStore wraps a *gorm.DB.
func NewGormOutboxStore(db *gorm.DB) *GormOutboxStore {
	return &GormOutboxStore{DB: db}
}

// MigrateOutbox creates the outbox table if needed.
func MigrateOutbox(db *gorm.DB) error {
	return db.AutoMigrate(&OutboxEvent{})
}

// Enqueue implements OutboxStore.
func (s *GormOutboxStore) Enqueue(ctx context.Context, ev CloudEvent) (uint, error) {
	payload, err := MarshalCloudEvent(ev)
	if err != nil {
		return 0, fmt.Errorf("events: outbox marshal: %w", err)
	}
	row := OutboxEvent{
		EventType: ev.Type,
		Source:    ev.Source,
		Subject:   ev.Subject,
		Payload:   string(payload),
		Status:    OutboxPending,
	}
	if err := s.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, fmt.Errorf("events: outbox enqueue: %w", err)
	}
	return row.ID, nil
}

// Pending implements OutboxStore. It claims the rows it returns: they move
// from pending to inflight inside one transaction, so two relays polling the
// same table (every service shares the ONE omoikane outbox, and a service may
// run more than one replica) can never publish the same row twice. Rows whose
// relay died mid-batch are reclaimed once OutboxInflightTimeout elapses.
func (s *GormOutboxStore) Pending(ctx context.Context, limit int) ([]OutboxEvent, error) {
	db := s.DB.WithContext(ctx)

	// Crash recovery: release claims that outlived their publisher.
	cutoff := time.Now().Add(-OutboxInflightTimeout)
	if err := db.Model(&OutboxEvent{}).
		Where("status = ? AND updated_at < ?", OutboxInflight, cutoff).
		Updates(map[string]any{"status": OutboxPending}).Error; err != nil {
		return nil, fmt.Errorf("events: outbox reclaim: %w", err)
	}

	var rows []OutboxEvent
	err := db.Transaction(func(tx *gorm.DB) error {
		// SKIP LOCKED: concurrent relays walk past each other's claimed rows
		// instead of blocking or re-taking them.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", OutboxPending).
			Order("id ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]uint, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		if err := tx.Model(&OutboxEvent{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"status": OutboxInflight}).Error; err != nil {
			return err
		}
		for i := range rows {
			rows[i].Status = OutboxInflight
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("events: outbox claim: %w", err)
	}
	return rows, nil
}

// MarkSent implements OutboxStore.
func (s *GormOutboxStore) MarkSent(ctx context.Context, id uint) error {
	now := time.Now()
	res := s.DB.WithContext(ctx).Model(&OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     OutboxSent,
			"sent_at":    &now,
			"last_error": "",
			"updated_at": now,
		})
	if res.Error != nil {
		return fmt.Errorf("events: outbox mark sent: %w", res.Error)
	}
	return nil
}

// MarkAttempt implements OutboxStore.
func (s *GormOutboxStore) MarkAttempt(ctx context.Context, id uint, err error, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	// Load current attempts so concurrent relay polls don't undercount.
	var row OutboxEvent
	if err := s.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return fmt.Errorf("events: outbox load for attempt: %w", err)
	}
	attempts := row.Attempts + 1
	status := OutboxPending
	if attempts >= maxAttempts {
		status = OutboxFailed
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	now := time.Now()
	res := s.DB.WithContext(ctx).Model(&OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"attempts":   attempts,
			"last_error": msg,
			"updated_at": now,
		})
	if res.Error != nil {
		return fmt.Errorf("events: outbox mark attempt: %w", res.Error)
	}
	return nil
}
