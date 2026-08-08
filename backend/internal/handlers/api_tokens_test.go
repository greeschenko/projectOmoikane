package handlers_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func tokenServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /users", h.Admin(h.GetUsers))
	mux.HandleFunc("GET /api-tokens", h.Admin(h.GetApiTokens))
	mux.HandleFunc("POST /api-tokens", h.Admin(h.CreateApiToken))
	mux.HandleFunc("DELETE /api-tokens/{id}", h.Admin(h.DeleteApiToken))
	return httptest.NewServer(mux)
}

func createTokenViaAPI(t *testing.T, server *httptest.Server, cookie *http.Cookie, role string) (uint, string) {
	t.Helper()
	body := fmt.Sprintf(`{"name":"CI Token","role":"%s","expiresInDays":30}`, role)
	resp := authenticatedRequest(t, "POST", server.URL+"/api-tokens", body, cookie)
	if resp.StatusCode != 201 {
		t.Fatalf("Create token failed (status %d): %s", resp.StatusCode, readBody(t, resp))
	}
	data := decodeJSON(t, readBody(t, resp))
	id := uint(data["id"].(float64))
	raw := data["token"].(string)
	return id, raw
}

func TestCreateApiToken_AdminCreatesAndUses(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	user := createTestUser(db, "Admin", "admin@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "admin@test.com", "pass123")

	_, raw := createTokenViaAPI(t, server, cookie, "admin")

	// Use the token on an admin-only endpoint
	req, _ := http.NewRequest("GET", server.URL+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("Token auth failed (status %d): %s", resp.StatusCode, readBody(t, resp))
	}
	_ = user
}

func TestApiToken_InvalidTokenRejected(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/users", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestApiToken_ExpiredTokenRejected(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "admin@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "admin@test.com", "pass123")
	_, raw := createTokenViaAPI(t, server, cookie, "admin")

	// Force the token to be expired in the DB
	hash := tokenHash(raw)
	if err := db.Model(&models.ApiToken{}).Where("token_hash = ?", hash).
		Update("expires_at", time.Now().Add(-1*time.Hour)).Error; err != nil {
		t.Fatalf("Failed to expire token: %v", err)
	}

	req, _ := http.NewRequest("GET", server.URL+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("Expected 401 for expired token, got %d", resp.StatusCode)
	}
}

func TestApiToken_RevokedTokenRejected(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "admin@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "admin@test.com", "pass123")
	id, raw := createTokenViaAPI(t, server, cookie, "admin")

	// Revoke via API
	delResp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/api-tokens/%d", server.URL, id), "", cookie)
	if delResp.StatusCode != 200 {
		t.Fatalf("Revoke failed (status %d)", delResp.StatusCode)
	}
	delResp.Body.Close()

	req, _ := http.NewRequest("GET", server.URL+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("Expected 401 for revoked token, got %d", resp.StatusCode)
	}
}

func TestApiToken_UserRoleTokenDeniedOnAdmin(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "admin@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "admin@test.com", "pass123")
	_, raw := createTokenViaAPI(t, server, cookie, "user")

	req, _ := http.NewRequest("GET", server.URL+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("Expected 403 for user-role token on admin route, got %d", resp.StatusCode)
	}
}

func TestApiToken_NonAdminCannotCreateToken(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	createTestUser(db, "User", "user@test.com", "pass123", "user")
	cookie := loginAs(t, server, "user@test.com", "pass123")

	resp := authenticatedRequest(t, "POST", server.URL+"/api-tokens", `{"name":"x"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("Expected 403 for non-admin token creation, got %d", resp.StatusCode)
	}
}

func TestApiToken_ListDoesNotExposeHashes(t *testing.T) {
	db := setupTestDB(t)
	server := tokenServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "admin@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "admin@test.com", "pass123")
	createTokenViaAPI(t, server, cookie, "admin")

	resp := authenticatedRequest(t, "GET", server.URL+"/api-tokens", "", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("List failed (status %d)", resp.StatusCode)
	}
	body := readBody(t, resp)
	if strings.Contains(body, "token") && strings.Contains(body, "hash") {
		// Only the meta keys matter; ensure no raw hash values leak
		var arr []map[string]interface{}
		if err := json.Unmarshal([]byte(body), &arr); err == nil && len(arr) > 0 {
			if _, exists := arr[0]["tokenHash"]; exists {
				t.Fatal("tokenHash leaked in list response")
			}
			if _, exists := arr[0]["hash"]; exists {
				t.Fatal("hash leaked in list response")
			}
		}
	}
}
