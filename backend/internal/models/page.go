package models

import "gorm.io/gorm"

type Page struct {
	gorm.Model
	Title           string `gorm:"not null"`
	Slug            string `gorm:"not null"`
	Content         string `gorm:"type:text"`
	MetaTitle       string
	MetaDescription string
	MetaKeywords    string
	ParentID        *uint
	SortOrder       int    `gorm:"default:0"`
	Status          string `gorm:"default:draft;not null"`
	InMenu          bool   `gorm:"default:false"`
	PreviewToken    string `gorm:"uniqueIndex;not null"`
}
