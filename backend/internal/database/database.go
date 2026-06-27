package database

import (
	"fmt"
	"log"

	"omoikane-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Database connected")
	DB = db
	return db, nil
}

func MustConnect(dsn string) *gorm.DB {
	db, err := Connect(dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	return db
}

func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.User{},
		&models.Page{},
		&models.BlogPost{},
		&models.Tag{},
		&models.BlogPostTag{},
		&models.Category{},
		&models.Like{},
		&models.MediaItem{},
		&models.Message{},
		&models.SiteSetting{},
	)
	if err != nil {
		return fmt.Errorf("AutoMigrate failed: %w", err)
	}
	log.Println("AutoMigrate completed")
	return nil
}

func MustAutoMigrate(db *gorm.DB) {
	if err := AutoMigrate(db); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}
}
