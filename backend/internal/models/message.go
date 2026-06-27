package models

import "gorm.io/gorm"

type Message struct {
	gorm.Model
	Title   string `gorm:"not null"`
	Content string `gorm:"type:text"`
	ReadBy  string `gorm:"type:text"`
}
