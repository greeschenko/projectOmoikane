package models

import "gorm.io/gorm"

type ContactMessage struct {
	gorm.Model
	Name    string `gorm:"not null"`
	Email   string `gorm:"not null"`
	Subject string `gorm:"not null"`
	Message string `gorm:"type:text;not null"`
	Read    bool   `gorm:"default:false"`
}
