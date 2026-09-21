package events

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// OutboxStatus values.
const (
	OutboxPending = "pending" // awaiting relay publish
	OutboxSent    = "sent"    // relay published to Kafka
	OutboxFailed  = "failed"  // exhausted OutboxMaxAttempts; operator attention
)

// OutboxEvent is the transactional-outbox row. Services append here IN THE
// SAME DB TRANSACTION as the business write (GORM tx), giving atomicity:
// either the business change and its event both commit, or neither does.
// A relay worker polls pending rows and publishes them to Kafka, then marks
// them sent. Consumers must tolerate at-least-once semantics.
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

	// Pending returns up to limit rows still to be published (oldest first).
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

// Pending implements OutboxStore.
func (s *GormOutboxStore) Pending(ctx context.Context, limit int) ([]OutboxEvent, error) {
	var rows []OutboxEvent
	err := s.DB.WithContext(ctx).
		Where("status = ?", OutboxPending).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("events: outbox pending: %w", err)
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
