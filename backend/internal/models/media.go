package models

import "gorm.io/gorm"

type MediaItem struct {
	gorm.Model
	Filename string `gorm:"not null"`
	MimeType string `gorm:"not null"`
	Size     int64  `gorm:"not null"`
	FilePath string `gorm:"not null"`
}
