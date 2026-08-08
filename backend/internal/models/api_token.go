package models

import (
	"time"

	"gorm.io/gorm"
)

// ApiToken authenticates programmatic clients (headless CMS mode).
// Only the SHA-256 hash of the raw token is stored.
type ApiToken struct {
	gorm.Model
	Name        string     `gorm:"not null"`
	TokenHash   string     `gorm:"uniqueIndex;not null"`
	Role        string     `gorm:"default:user;not null"`
	Description string
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
}
