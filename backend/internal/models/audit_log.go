package models

import "gorm.io/gorm"

type AuditLog struct {
	gorm.Model
	UserID    uint   `gorm:"index"`
	UserName  string `gorm:"not null"`
	Action    string `gorm:"not null;index"`
	EntityType string `gorm:"not null;index"`
	EntityID   uint
	Detail    string
	IP        string
	UserAgent string
}
