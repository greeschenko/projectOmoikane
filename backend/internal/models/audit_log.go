package models

import (
	"time"

	"gorm.io/gorm"
)

// AuditLog records a single domain event for the admin audit trail. Fields
// carry explicit camelCase JSON tags: the admin UI (frontend/app/admin/
// audit-logs/page.tsx) reads camelCase keys, and Go's default serialization
// of struct fields would otherwise emit capitalized keys that match nothing
// on the frontend.
type AuditLog struct {
	// gorm.Model is intentionally not embedded here — an embedded struct's
	// fields cannot carry per-field json tags. GORM still detects ID /
	// CreatedAt / UpdatedAt / DeletedAt by convention, so soft-delete and
	// auto-timestamps behave exactly like gorm.Model.
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
	// EventID is the CloudEvent id this row was derived from. Unique so the
	// Kafka consumer is idempotent under at-least-once redelivery.
	EventID    string `gorm:"uniqueIndex" json:"eventId"`
	UserID     uint   `gorm:"index" json:"userId"`
	UserName   string `gorm:"not null" json:"userName"`
	Action     string `gorm:"not null;index" json:"action"`
	EntityType string `gorm:"not null;index" json:"entityType"`
	EntityID   uint   `json:"entityId"`
	Detail     string `json:"detail"`
	IP         string `json:"ip"`
	UserAgent  string `json:"userAgent"`
}
