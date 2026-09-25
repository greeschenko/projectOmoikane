package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"omoikane-backend/internal/auth"
	"omoikane-backend/internal/database"
	"omoikane-backend/internal/events"
	"omoikane-backend/internal/handlers"
	"omoikane-backend/internal/models"
	"omoikane-backend/internal/observability"

	"gorm.io/gorm"
)

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func webhooksTestDSN() string {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable"
	}
	return dsn
}

// setupWebhooksService boots the real webhooks-service mux (newWebhooksMux)
// against the test database, mirroring production (admin-JWT validation only).
func setupWebhooksService(t *testing.T) (*gorm.DB, *httptest.Server) {
	t.Helper()
	db := connectCleanTestDB(t)
	h := &handlers.Handler{DB: db, JWTSecret: "test-secret"}
	s := httptest.NewServer(newWebhooksMux(h))
	t.Cleanup(s.Close)
	return db, s
}

func connectCleanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Connect(webhooksTestDSN())
	if err != nil {
		t.Fatalf("connect test DB: %v", err)
	}
	if sqlDB, serr := db.DB(); serr == nil {
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(3)
	}
	t.Cleanup(func() {
		if sqlDB, serr := db.DB(); serr == nil {
			sqlDB.Close()
		}
	})
	if err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE").Error; err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test DB: %v", err)
	}
	return db
}

// sessionCookie mints a JWT and returns it as the "session" cookie.
func sessionCookie(t *testing.T, userID uint, role string) map[string][]string {
	t.Helper()
	token, err := auth.GenerateToken(userID, role, "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	return map[string][]string{"Cookie": {"session=" + token}}
}

func doReq(t *testing.T, method, url, body string, headers map[string][]string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func TestWebhooksService_Health(t *testing.T) {
	_, s := setupWebhooksService(t)

	resp := doReq(t, "GET", s.URL+"/health", "", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status = %q, want ok", body["status"])
	}
}

// TestWebhooksService_AdminGate: every /webhooks* route requires an admin;
// anonymous 401, non-admin 403, admin 200.
func TestWebhooksService_AdminGate(t *testing.T) {
	db, s := setupWebhooksService(t)
	user := models.User{Name: "U", Email: "u@t.com", Password: "x", Role: "user", Status: "active"}
	admin := models.User{Name: "A", Email: "a@t.com", Password: "x", Role: "admin", Status: "active"}
	db.Create(&user)
	db.Create(&admin)

	routes := []string{"/webhooks", "/webhooks/deliveries"}
	for _, route := range routes {
		resp := doReq(t, "GET", s.URL+route, "", nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("anonymous GET %s: expected 401, got %d", route, resp.StatusCode)
		}
		resp = doReq(t, "GET", s.URL+route, "", sessionCookie(t, user.ID, "user"))
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("user GET %s: expected 403, got %d", route, resp.StatusCode)
		}
		resp = doReq(t, "GET", s.URL+route, "", sessionCookie(t, admin.ID, "admin"))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("admin GET %s: expected 200, got %d", route, resp.StatusCode)
		}
	}
}

func TestWebhooksService_CRUD(t *testing.T) {
	db, s := setupWebhooksService(t)
	admin := models.User{Name: "A", Email: "a@t.com", Password: "x", Role: "admin", Status: "active"}
	db.Create(&admin)
	cookie := sessionCookie(t, admin.ID, "admin")

	body := fmt.Sprintf(`{"eventType":"%s","url":"http://example.com/hook"}`, events.TypePostPublished)
	resp := doReq(t, "POST", s.URL+"/webhooks", body, cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d (%s)", resp.StatusCode, readBody(resp))
	}
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if created["secret"] == "" {
		t.Fatal("expected one-time secret in create response")
	}
	if _, leaked := created["secret"]; leaked {
		// covered: secret present once at creation
	}
	id := uint(created["id"].(float64))

	// List must NOT echo the secret.
	listResp := doReq(t, "GET", s.URL+"/webhooks", "", cookie)
	defer listResp.Body.Close()
	listBody := readBody(listResp)
	if strings.Contains(listBody, created["secret"].(string)) {
		t.Fatal("list response leaked the secret")
	}

	// Update: toggle + rotate.
	putResp := doReq(t, "PUT", s.URL+"/webhooks/"+fmt.Sprint(id), `{"active":false}`, cookie)
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("update: expected 200, got %d", putResp.StatusCode)
	}
	var sub models.WebhookSubscription
	if err := db.First(&sub, id).Error; err != nil {
		t.Fatalf("load sub: %v", err)
	}
	if sub.Active {
		t.Fatal("expected inactive after update")
	}

	// Delete.
	delResp := doReq(t, "DELETE", s.URL+"/webhooks/"+fmt.Sprint(id), "", cookie)
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", delResp.StatusCode)
	}
}

func TestWebhooksService_TestPing_EnqueuesDelivery(t *testing.T) {
	db, s := setupWebhooksService(t)
	admin := models.User{Name: "A", Email: "a@t.com", Password: "x", Role: "admin", Status: "active"}
	db.Create(&admin)
	cookie := sessionCookie(t, admin.ID, "admin")

	sub := models.WebhookSubscription{EventType: events.TypeContactReceived, URL: "http://example.com/hook", Secret: "k", Active: true}
	db.Create(&sub)

	resp := doReq(t, "POST", s.URL+"/webhooks/"+fmt.Sprint(sub.ID)+"/test", "", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("test ping: expected 201, got %d (%s)", resp.StatusCode, readBody(resp))
	}

	var d models.WebhookDelivery
	if err := db.Where("subscription_id = ?", sub.ID).First(&d).Error; err != nil {
		t.Fatalf("expected delivery row: %v", err)
	}
	if d.Status != statusPending {
		t.Fatalf("expected pending, got %q", d.Status)
	}
	if !strings.Contains(d.Payload, handlers.TypeWebhookPing) {
		t.Fatalf("ping payload missing event type: %s", d.Payload)
	}
}

func TestWebhooksService_MetricsEndpoint(t *testing.T) {
	db := connectCleanTestDB(t)
	h := &handlers.Handler{DB: db, JWTSecret: "test-secret"}
	// Production-faithful: the real server wraps the mux in observability.Middleware.
	mux := observability.Middleware(newWebhooksMux(h))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Warm the middleware with a request that renders a counter (health is
	// public and route-normalizes to /health).
	resp := doReq(t, "GET", srv.URL+"/health", "", nil)
	resp.Body.Close()

	metricsResp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer metricsResp.Body.Close()
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", metricsResp.StatusCode)
	}
	text := readBody(metricsResp)
	for _, want := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_in_flight_requests",
		"route=\"/health\"",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("metrics missing %q", want)
		}
	}
}

// ---------------------------------------------------------------- consumer --

// aCloudEvent builds a minimal CloudEvent for consumer-handler tests.
func aCloudEvent(id, eventType string, data interface{}) events.CloudEvent {
	raw, _ := json.Marshal(data)
	return events.CloudEvent{
		SpecVersion: "1.0",
		ID:          id,
		Source:      "content-service",
		Type:        eventType,
		Time:        time.Now().UTC(),
		Data:        raw,
	}
}

func TestConsumerHandler_MatchesSubscriptions(t *testing.T) {
	db := connectCleanTestDB(t)
	h := &webhookEventHandler{db: db}

	db.Create(&models.WebhookSubscription{EventType: events.TypePostPublished, URL: "http://a", Active: true, Secret: "s1"})
	db.Create(&models.WebhookSubscription{EventType: events.TypePostPublished, URL: "http://b", Active: false, Secret: ""})
	db.Create(&models.WebhookSubscription{EventType: events.TypeMediaUploaded, URL: "http://c", Active: true})

	ev := aCloudEvent("ev-post-1", events.TypePostPublished, map[string]interface{}{"id": 7})
	if err := h.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var rows []models.WebhookDelivery
	db.Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("expected 1 delivery (only the active matching sub), got %d", len(rows))
	}
	if rows[0].SubscriptionID != 1 {
		t.Fatalf("expected delivery for sub 1, got sub %d", rows[0].SubscriptionID)
	}
	if !strings.Contains(rows[0].Payload, ev.ID) {
		t.Fatalf("payload does not carry the event id: %s", rows[0].Payload)
	}
}

func TestConsumerHandler_RedeliveryIsIdempotent(t *testing.T) {
	db := connectCleanTestDB(t)
	h := &webhookEventHandler{db: db}
	db.Create(&models.WebhookSubscription{EventType: events.TypePagePublished, URL: "http://a", Active: true})

	ev := aCloudEvent("ev-page-1", events.TypePagePublished, map[string]interface{}{"id": 3})
	if err := h.Handle(context.Background(), ev); err != nil {
		t.Fatalf("first Handle: %v", err)
	}
	// At-least-once redelivery: same event again must not duplicate rows.
	if err := h.Handle(context.Background(), ev); err != nil {
		t.Fatalf("second Handle: %v", err)
	}

	var n int64
	db.Model(&models.WebhookDelivery{}).Count(&n)
	if n != 1 {
		t.Fatalf("expected exactly 1 delivery after redelivery, got %d", n)
	}
}

func TestConsumerHandler_NoSubscribersAcksFast(t *testing.T) {
	db := connectCleanTestDB(t)
	h := &webhookEventHandler{db: db}

	ev := aCloudEvent("ev-none", events.TypeSettingsUpdated, map[string]interface{}{})
	if err := h.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	var n int64
	db.Model(&models.WebhookDelivery{}).Count(&n)
	if n != 0 {
		t.Fatalf("expected no deliveries without matching subs, got %d", n)
	}
}

// ------------------------------------------------------------------- pump --

// fakeSink records signature + body of received webhook POSTs at /hook.
type fakeSink struct {
	mu        sync.Mutex
	signature string
	body      string
	received  int
}

func (f *fakeSink) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /hook", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.signature = r.Header.Get("X-Omoikane-Signature")
		b, _ := io.ReadAll(r.Body)
		f.body = string(b)
		f.received++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	return mux
}

// closedPort returns a port that nothing listens on (refused connections).
func closedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestDeliveryPump_DeliversWithSignature(t *testing.T) {
	db := connectCleanTestDB(t)
	sink := &fakeSink{}
	ss := httptest.NewServer(sink.handler())
	defer ss.Close()

	sub := models.WebhookSubscription{EventType: events.TypePostPublished, URL: ss.URL + "/hook", Secret: "hunter2", Active: true}
	db.Create(&sub)
	db.Create(&models.WebhookDelivery{
		SubscriptionID: sub.ID, EventID: "ev-1", EventType: events.TypePostPublished,
		Payload: `{"specversion":"1.0","id":"ev-1",` +
			`"type":"org.omoikane.content.post.published.v1","source":"content-service",` +
			`"time":"2026-01-01T00:00:00Z","data":{"id":1,"title":"Hi"}}`,
		Status: statusPending, NextAttemptAt: time.Now().UTC(),
	})

	w := newDeliveryWorker(db)
	if n, err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	} else if n != 1 {
		t.Fatalf("expected 1 delivery attempted, got %d", n)
	}

	// Row is delivered with the recorded HTTP status.
	var d models.WebhookDelivery
	db.First(&d)
	if d.Status != statusDelivered {
		t.Fatalf("expected delivered, got %q", d.Status)
	}
	if d.HTTPStatus != http.StatusOK {
		t.Fatalf("expected http 200, got %d", d.HTTPStatus)
	}
	if d.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", d.Attempts)
	}

	// Sink saw the exact body + a valid HMAC signature.
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if sink.received != 1 {
		t.Fatalf("expected 1 sink hit, got %d", sink.received)
	}
	mac := hmac.New(sha256.New, []byte("hunter2"))
	mac.Write([]byte(sink.body))
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if sink.signature != want {
		t.Fatalf("signature = %q, want %q", sink.signature, want)
	}
}

func TestDeliveryPump_RetriesThenExpires(t *testing.T) {
	db := connectCleanTestDB(t)

	sub := models.WebhookSubscription{EventType: events.TypeAuthLogin, URL: "http://127.0.0.1:" + fmt.Sprint(closedPort(t)) + "/nope", Active: true}
	db.Create(&sub)
	db.Create(&models.WebhookDelivery{
		SubscriptionID: sub.ID, EventID: "ev-fail", EventType: events.TypeAuthLogin,
		Payload: `{"specversion":"1.0","id":"ev-fail"}`, Status: statusPending,
		NextAttemptAt: time.Now().UTC(),
	})

	w := newDeliveryWorker(db)
	// Fast expiry for the test: 3 total attempts.
	w.maxAttempts = 3

	run := func(attempt int) {
		if n, err := w.RunOnce(context.Background()); err != nil {
			t.Fatalf("RunOnce #%d: %v", attempt, err)
		} else if n != 1 {
			t.Fatalf("RunOnce #%d: expected 1 attempted row, got %d", attempt, n)
		}
	}

	// Attempt 1 -> failed, retry scheduled with backoff.
	run(1)
	var d models.WebhookDelivery
	db.First(&d)
	if d.Status != statusFailed {
		t.Fatalf("after attempt 1: expected failed, got %q", d.Status)
	}
	if d.Attempts != 1 {
		t.Fatalf("after attempt 1: expected attempts=1, got %d", d.Attempts)
	}
	if !d.NextAttemptAt.After(time.Now().Add(500 * time.Millisecond)) {
		t.Fatalf("expected a future retry schedule, got %v", d.NextAttemptAt)
	}
	if d.Error == "" {
		t.Fatal("expected a recorded error on failure")
	}

	// Force the retry window open and run again: attempt 2.
	db.Model(&d).Update("next_attempt_at", time.Now().Add(-time.Second))
	run(2)
	db.First(&d)
	if d.Attempts != 2 {
		t.Fatalf("after attempt 2: expected attempts=2, got %d", d.Attempts)
	}

	// Attempt 3 = maxAttempts -> terminal expired (the "DLQ" state).
	db.Model(&d).Update("next_attempt_at", time.Now().Add(-time.Second))
	run(3)
	db.First(&d)
	if d.Status != statusExpired {
		t.Fatalf("after attempt 3: expected expired, got %q", d.Status)
	}
	if d.Attempts != 3 {
		t.Fatalf("expected 3 attempts recorded, got %d", d.Attempts)
	}

	// Expired rows are never selected again.
	n, err := w.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce after expiry: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected no rows after expiry, got %d", n)
	}
}

func TestDeliveryPump_MissingSubscriptionExpires(t *testing.T) {
	db := connectCleanTestDB(t)
	db.Create(&models.WebhookDelivery{
		SubscriptionID: 999, EventID: "ev-orphan", EventType: events.TypeMediaUploaded,
		Payload: `{"id":"ev-orphan"}`, Status: statusPending, NextAttemptAt: time.Now().UTC(),
	})

	w := newDeliveryWorker(db)
	if n, _ := w.RunOnce(context.Background()); n != 1 {
		t.Fatalf("expected the orphan row attempted, got %d", n)
	}
	var d models.WebhookDelivery
	db.First(&d)
	if d.Status != statusExpired {
		t.Fatalf("expected expired for missing subscription, got %q", d.Status)
	}
}

func TestSignPayload(t *testing.T) {
	got := signPayload("k", []byte("hello"))
	want := "sha256=" + hex.EncodeToString(hmacKey("k", []byte("hello")))
	if got != want {
		t.Fatalf("signPayload = %q, want %q", got, want)
	}
}

func hmacKey(secret string, body []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return mac.Sum(nil)
}

func TestBackoffSchedule(t *testing.T) {
	w := newDeliveryWorker(nil)
	w.baseBackoff = time.Second
	w.backoffCap = 60 * time.Second
	want := []time.Duration{1, 2, 4, 8, 16, 32, 60, 60}
	for i, exp := range want {
		if got := w.backoff(i + 1); got != exp*time.Second {
			t.Errorf("backoff(attempt %d) = %v, want %v", i+1, got, exp*time.Second)
		}
	}
}
