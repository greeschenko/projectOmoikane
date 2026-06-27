package models

import (
	"time"

	"gorm.io/gorm"
)

type BlogPost struct {
	gorm.Model
	Title         string     `gorm:"not null"`
	Slug          string     `gorm:"uniqueIndex;not null"`
	Content       string     `gorm:"type:text"`
	Excerpt       string     `gorm:"type:text"`
	AuthorID      uint       `gorm:"not null;index"`
	Status        string     `gorm:"default:draft;not null"`
	PublishDate   *time.Time
	FeaturedImage string
	LikeCount     int `gorm:"default:0"`
	CategoryID    *uint
}
