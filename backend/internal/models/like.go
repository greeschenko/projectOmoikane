package models

import "time"

type Like struct {
	BlogPostID uint `gorm:"primaryKey"`
	UserID     uint `gorm:"primaryKey"`
	CreatedAt  time.Time
}
