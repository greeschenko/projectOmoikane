package handlers_test

import (
	"fmt"
	"testing"

	"omoikane-backend/internal/models"
)

func TestGetUsers_AdminCanList(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	createTestUser(db, "User1", "user1@test.com", "UserPass1!", "user")
	createTestUser(db, "User2", "user2@test.com", "UserPass2!", "user")

	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")
	resp := authenticatedRequest(t, "GET", s.URL+"/users", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCreateUser_AdminCreatesUser(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"name":"New User","email":"new@test.com","password":"UserPass1!","role":"user"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/users", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.User{}).Where("email = ?", "new@test.com").Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestCreateUser_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	resp := authenticatedRequest(t, "POST", s.URL+"/users", `{}`, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	createTestUser(db, "Existing", "existing@test.com", "Pass1234!", "user")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"name":"Dup","email":"existing@test.com","password":"Pass1234!"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/users", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateUser_ShortPassword(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"name":"New","email":"new@test.com","password":"abc"}`
	resp := authenticatedRequest(t, "POST", s.URL+"/users", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for short password, got %d", resp.StatusCode)
	}
}

func TestUpdateUser_AdminUpdatesUser(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	_ = createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	target := createTestUser(db, "Target", "target@test.com", "Pass1234!", "user")

	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"name":"Updated Name","role":"admin"}`
	url := s.URL + "/users/" + itoa(target.ID)
	resp := authenticatedRequest(t, "PUT", url, body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var updated models.User
	db.First(&updated, target.ID)
	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", updated.Name)
	}
	if updated.Role != "admin" {
		t.Errorf("Expected role 'admin', got %s", updated.Role)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	body := `{"name":"Nobody"}`
	resp := authenticatedRequest(t, "PUT", s.URL+"/users/99999", body, cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteUser_AdminDeletesUser(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	target := createTestUser(db, "Target", "target@test.com", "Pass1234!", "user")

	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")
	url := s.URL + "/users/" + itoa(target.ID)
	resp := authenticatedRequest(t, "DELETE", url, "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var count int64
	db.Model(&models.User{}).Where("id = ?", target.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected user to be deleted, count=%d", count)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	s := userServer(db)
	defer s.Close()

	createTestUser(db, "Admin", "admin@test.com", "AdminPass1!", "admin")
	cookie := loginAs(t, s, "admin@test.com", "AdminPass1!")

	resp := authenticatedRequest(t, "DELETE", s.URL+"/users/99999", "", cookie)
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func itoa(n uint) string {
	return fmt.Sprintf("%d", n)
}
