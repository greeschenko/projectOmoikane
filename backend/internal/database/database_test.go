package database_test

import (
	"os"
	"testing"

	"omoikane-backend/internal/database"

	"gorm.io/gorm"
)

func getTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

func connectAndClean(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := getTestDSN()
	db, err := database.Connect(dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}
	resetSchema(t, db)
	return db
}

func resetSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE").Error; err != nil {
		t.Fatalf("Failed to drop public schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("Failed to create public schema: %v", err)
	}
}

func TestDatabase_Connect_Success(t *testing.T) {
	dsn := getTestDSN()
	db, err := database.Connect(dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestDatabase_AutoMigrate_CreatesTables(t *testing.T) {
	db := connectAndClean(t)

	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// Verify all expected tables exist
	expectedTables := []string{
		"users", "pages", "blog_posts", "tags", "blog_post_tags",
		"categories", "likes", "media_items", "messages", "site_settings",
	}
	for _, table := range expectedTables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("Expected table %q to exist after AutoMigrate", table)
		}
	}
}

func TestDatabase_AutoMigrate_Idempotent(t *testing.T) {
	db := connectAndClean(t)

	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("First AutoMigrate failed: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("Second AutoMigrate failed: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("Third AutoMigrate failed: %v", err)
	}

	if !db.Migrator().HasTable("users") {
		t.Error("Expected users table to exist after repeated AutoMigrate")
	}
}
