package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"

	"gorm.io/gorm"
)

// webhookServer wires the admin webhook handlers behind login (the service mux
// itself is exercised in cmd/webhooks tests).
func webhookServer(db *gorm.DB) *httptest.Server {
	h := &handlers.Handler{DB: db, JWTSecret: testJWTSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /webhooks", h.Admin(h.GetWebhooks))
	mux.HandleFunc("POST /webhooks", h.Admin(h.CreateWebhook))
	mux.HandleFunc("GET /webhooks/deliveries", h.Admin(h.GetWebhookDeliveries))
	mux.HandleFunc("GET /webhooks/{id}", h.Admin(h.GetWebhook))
	mux.HandleFunc("PUT /webhooks/{id}", h.Admin(h.UpdateWebhook))
	mux.HandleFunc("DELETE /webhooks/{id}", h.Admin(h.DeleteWebhook))
	mux.HandleFunc("POST /webhooks/{id}/test", h.Admin(h.TestWebhook))
	return httptest.NewServer(mux)
}

func createWebhookViaAPI(t *testing.T, server *httptest.Server, cookie *http.Cookie, eventType, url string) map[string]interface{} {
	t.Helper()
	body := fmt.Sprintf(`{"eventType":"%s","url":"%s"}`, eventType, url)
	resp := authenticatedRequest(t, "POST", server.URL+"/webhooks", body, cookie)
	if resp.StatusCode != 201 {
		t.Fatalf("Create webhook failed (status %d): %s", resp.StatusCode, readBody(t, resp))
	}
	data := decodeJSON(t, readBody(t, resp))
	if _, ok := data["secret"]; !ok {
		t.Fatal("create response did not include the one-time secret")
	}
	return data
}

func TestWebhooks_AdminOnly(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	createTestUser(db, "User", "u@test.com", "pass123", "user")

	// Anonymous: 401
	resp := authenticatedRequest(t, "GET", server.URL+"/webhooks", "", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous GET /webhooks: expected 401, got %d", resp.StatusCode)
	}

	// Non-admin: 403
	userCookie := loginAs(t, server, "u@test.com", "pass123")
	resp = authenticatedRequest(t, "GET", server.URL+"/webhooks", "", userCookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("user GET /webhooks: expected 403, got %d", resp.StatusCode)
	}

	// Admin: 200
	adminCookie := loginAs(t, server, "a@test.com", "pass123")
	resp = authenticatedRequest(t, "GET", server.URL+"/webhooks", "", adminCookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin GET /webhooks: expected 200, got %d", resp.StatusCode)
	}
}

func TestCreateWebhook_RejectsUnknownEventType(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "a@test.com", "pass123")

	resp := authenticatedRequest(t, "POST", server.URL+"/webhooks",
		`{"eventType":"org.omoikane.custom.v1","url":"http://example.com/hook"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown event type, got %d", resp.StatusCode)
	}

	resp = authenticatedRequest(t, "POST", server.URL+"/webhooks",
		`{"eventType":"","url":"http://example.com/hook"}`, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty event type, got %d", resp.StatusCode)
	}
}

func TestCreateWebhook_RejectsEmptyURL(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "a@test.com", "pass123")

	resp := authenticatedRequest(t, "POST", server.URL+"/webhooks",
		fmt.Sprintf(`{"eventType":"%s","url":""}`, events.TypePostPublished), cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty URL, got %d", resp.StatusCode)
	}
}

func TestWebhookCRUD(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "a@test.com", "pass123")

	created := createWebhookViaAPI(t, server, cookie, events.TypePostPublished, "http://example.com/hook")
	id := uint(created["id"].(float64))
	rawSecret := created["secret"].(string)
	if rawSecret == "" {
		t.Fatal("expected a generated secret")
	}

	// List: shows the subscription, NEVER the secret.
	listResp := authenticatedRequest(t, "GET", server.URL+"/webhooks", "", cookie)
	defer listResp.Body.Close()
	listBody := readBody(t, listResp)
	if !strings.Contains(listBody, events.TypePostPublished) {
		t.Fatalf("list missing subscription: %s", listBody)
	}
	if strings.Contains(listBody, rawSecret) {
		t.Fatal("list response leaked the webhook secret")
	}
	arr := decodeJSONArray(t, listBody)
	first := arr[0].(map[string]interface{})
	if _, hasSecret := first["secret"]; hasSecret {
		t.Fatal("subscription JSON serializer emits the secret")
	}

	// Get single.
	getResp := authenticatedRequest(t, "GET", fmt.Sprintf("%s/webhooks/%d", server.URL, id), "", cookie)
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /webhooks/{id}: expected 200, got %d", getResp.StatusCode)
	}

	// Update: toggle active + rotate secret.
	putResp := authenticatedRequest(t, "PUT", fmt.Sprintf("%s/webhooks/%d", server.URL, id),
		`{"active":false,"secret":"new-secret-key"}`, cookie)
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /webhooks/{id}: expected 200, got %d (%s)", putResp.StatusCode, readBody(t, putResp))
	}
	var sub models.WebhookSubscription
	if err := db.First(&sub, id).Error; err != nil {
		t.Fatalf("load sub: %v", err)
	}
	if sub.Active {
		t.Fatal("expected subscription deactivated after update")
	}
	if sub.Secret != "new-secret-key" {
		t.Fatalf("expected rotated secret, got %q", sub.Secret)
	}

	// 404 on missing id.
	missResp := authenticatedRequest(t, "GET", server.URL+"/webhooks/9999", "", cookie)
	defer missResp.Body.Close()
	if missResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d", missResp.StatusCode)
	}

	// Delete.
	delResp := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/webhooks/%d", server.URL, id), "", cookie)
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /webhooks/{id}: expected 204, got %d", delResp.StatusCode)
	}
	delResp2 := authenticatedRequest(t, "DELETE", fmt.Sprintf("%s/webhooks/%d", server.URL, id), "", cookie)
	defer delResp2.Body.Close()
	if delResp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", delResp2.StatusCode)
	}
}

func TestTestWebhook_EnqueuesPendingDelivery(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "a@test.com", "pass123")

	created := createWebhookViaAPI(t, server, cookie, events.TypeContactReceived, "http://example.com/hook")
	id := uint(created["id"].(float64))

	pingResp := authenticatedRequest(t, "POST", fmt.Sprintf("%s/webhooks/%d/test", server.URL, id), "", cookie)
	defer pingResp.Body.Close()
	if pingResp.StatusCode != http.StatusCreated {
		t.Fatalf("test ping: expected 201, got %d (%s)", pingResp.StatusCode, readBody(t, pingResp))
	}

	var d models.WebhookDelivery
	if err := db.Where("subscription_id = ?", id).First(&d).Error; err != nil {
		t.Fatalf("expected a pending delivery row: %v", err)
	}
	if d.Status != "pending" {
		t.Fatalf("expected pending status, got %q", d.Status)
	}
	if d.EventType != handlers.TypeWebhookPing {
		t.Fatalf("expected ping event type, got %q", d.EventType)
	}
	if !strings.Contains(d.Payload, `"org.omoikane.webhooks.ping.v1"`) {
		t.Fatalf("ping payload missing event type: %s", d.Payload)
	}
}

func TestTestWebhook_InactiveSubscriptionRejected(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "a@test.com", "pass123")

	createResp := authenticatedRequest(t, "POST", server.URL+"/webhooks",
		fmt.Sprintf(`{"eventType":"%s","url":"http://example.com/hook","active":false}`, events.TypeMediaUploaded), cookie)
	data := decodeJSON(t, readBody(t, createResp))
	id := uint(data["id"].(float64))

	pingResp := authenticatedRequest(t, "POST", fmt.Sprintf("%s/webhooks/%d/test", server.URL, id), "", cookie)
	defer pingResp.Body.Close()
	if pingResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("test ping on inactive sub: expected 400, got %d", pingResp.StatusCode)
	}
}

func TestGetWebhookDeliveries_Filters(t *testing.T) {
	db := setupTestDB(t)
	server := webhookServer(db)
	defer server.Close()

	createTestUser(db, "Admin", "a@test.com", "pass123", "admin")
	cookie := loginAs(t, server, "a@test.com", "pass123")

	created := createWebhookViaAPI(t, server, cookie, events.TypePostPublished, "http://example.com/hook")
	id := uint(created["id"].(float64))

	// Seed two delivery rows directly (one delivered, one pending).
	db.Create(&models.WebhookDelivery{
		SubscriptionID: id, EventID: "ev-1", EventType: events.TypePostPublished,
		Payload: `{"id":"ev-1"}`, Status: "delivered", Attempts: 1, HTTPStatus: 200,
	})
	db.Create(&models.WebhookDelivery{
		SubscriptionID: id, EventID: "ev-2", EventType: events.TypeMediaUploaded,
		Payload: `{"id":"ev-2"}`, Status: "pending", NextAttemptAt: time.Now().UTC(),
	})

	cases := []struct {
		query string
		want  int
	}{
		{"", 2},
		{"status=delivered", 1},
		{"eventType=" + events.TypeMediaUploaded, 1},
		{fmt.Sprintf("subscriptionId=%d", id), 2},
		{"status=bogus", 0},
	}
	for _, c := range cases {
		url := server.URL + "/webhooks/deliveries"
		if c.query != "" {
			url += "?" + c.query
		}
		resp := authenticatedRequest(t, "GET", url, "", cookie)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("deliveries query %q: expected 200, got %d", c.query, resp.StatusCode)
		}
		body := decodeJSON(t, readBody(t, resp))
		if total := int(body["total"].(float64)); total != c.want {
			t.Fatalf("deliveries query %q: expected total %d, got %d", c.query, c.want, total)
		}
		delivs := body["deliveries"].([]interface{})
		if len(delivs) != c.want {
			t.Fatalf("deliveries query %q: expected %d rows, got %d", c.query, c.want, len(delivs))
		}
	}
}