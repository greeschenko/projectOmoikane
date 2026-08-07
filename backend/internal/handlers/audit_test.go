package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func auditServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /audit-logs", h.Admin(h.GetAuditLogs))
	return httptest.NewServer(mux)
}

func createTestAuditLog(db *gorm.DB, userID uint, userName, action, entityType string, entityID uint, detail string) models.AuditLog {
	alog := models.AuditLog{
		UserID:     userID,
		UserName:   userName,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Detail:     detail,
	}
	db.Create(&alog)
	return alog
}

func TestGetAuditLogs_AdminCanList(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestUser(db, "Admin", "admin@test.com", "pass", "admin")

	createTestAuditLog(db, admin.ID, "Admin", "create", "user", 1, "Created user John")
	createTestAuditLog(db, admin.ID, "Admin", "delete", "page", 5, "Deleted page Home")

	server := auditServer(db)
	defer server.Close()

	cookie := loginAs(t, server, "admin@test.com", "pass")
	resp := authenticatedRequest(t, "GET", server.URL+"/audit-logs", "", cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	logs := result["logs"].([]interface{})
	if len(logs) != 2 {
		t.Fatalf("Expected 2 audit logs, got %d", len(logs))
	}

	total := result["total"].(float64)
	if total != 2 {
		t.Fatalf("Expected total 2, got %d", int(total))
	}
}

func TestGetAuditLogs_NonAdminRejected(t *testing.T) {
	db := setupTestDB(t)
	createTestUser(db, "User", "user@test.com", "pass", "user")

	server := auditServer(db)
	defer server.Close()

	cookie := loginAs(t, server, "user@test.com", "pass")
	resp := authenticatedRequest(t, "GET", server.URL+"/audit-logs", "", cookie)
	if resp.StatusCode != 403 {
		t.Fatalf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestGetAuditLogs_FilterByEntity(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestUser(db, "Admin", "admin@test.com", "pass", "admin")

	createTestAuditLog(db, admin.ID, "Admin", "create", "user", 1, "")
	createTestAuditLog(db, admin.ID, "Admin", "create", "page", 2, "")
	createTestAuditLog(db, admin.ID, "Admin", "delete", "user", 3, "")

	server := auditServer(db)
	defer server.Close()

	cookie := loginAs(t, server, "admin@test.com", "pass")
	resp := authenticatedRequest(t, "GET", server.URL+"/audit-logs?entity=user", "", cookie)

	body := readBody(t, resp)
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	logs := result["logs"].([]interface{})
	if len(logs) != 2 {
		t.Fatalf("Expected 2 user logs, got %d", len(logs))
	}
}

func TestGetAuditLogs_SearchByUserName(t *testing.T) {
	db := setupTestDB(t)
	admin := createTestUser(db, "Admin", "admin@test.com", "pass", "admin")

	createTestAuditLog(db, admin.ID, "Admin", "create", "user", 1, "")
	createTestAuditLog(db, 0, "System", "backup", "database", 0, "")

	server := auditServer(db)
	defer server.Close()

	cookie := loginAs(t, server, "admin@test.com", "pass")
	resp := authenticatedRequest(t, "GET", server.URL+"/audit-logs?search=Admin", "", cookie)

	body := readBody(t, resp)
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	logs := result["logs"].([]interface{})
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log for Admin search, got %d", len(logs))
	}
}
