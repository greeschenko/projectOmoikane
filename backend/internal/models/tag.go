package models

import "gorm.io/gorm"

type Tag struct {
	gorm.Model
	Name string `gorm:"uniqueIndex;not null"`
	Slug string `gorm:"uniqueIndex;not null"`
}

type BlogPostTag struct {
	BlogPostID uint `gorm:"primaryKey"`
	TagID      uint `gorm:"primaryKey"`
}
