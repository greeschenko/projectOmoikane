package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"omoikane-backend/internal/auth"
)

// fakeStatsUpstream serves canned /internal/stats payloads and records every
// request (method/path/token) for routing assertions.
type fakeStatsUpstream struct {
	server   *httptest.Server
	mu       sync.Mutex
	payload  map[string]interface{}
	requests []string
	tokens   []string
}

func newFakeStatsUpstream(payload map[string]interface{}) *fakeStatsUpstream {
	f := &fakeStatsUpstream{payload: payload}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/stats", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.URL.Path)
		f.tokens = append(f.tokens, r.Header.Get("X-Internal-Token"))
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(f.payload)
	})
	f.server = httptest.NewServer(mux)
	return f
}

func (f *fakeStatsUpstream) close() { f.server.Close() }

func (f *fakeStatsUpstream) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeStatsUpstream) token() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.tokens) == 0 {
		return ""
	}
	return f.tokens[0]
}

// adminCookie mints an admin JWT for the dashboard admin gate.
func adminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	token, err := auth.GenerateToken(1, "admin", "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	return &http.Cookie{Name: "session", Value: token}
}

func TestDashboardService_Health(t *testing.T) {
	mux := newDashboardMux(dashboardCfg{JWTSecret: "test-secret"})
	s := httptest.NewServer(mux)
	defer s.Close()

	resp, err := http.Get(s.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d", resp.StatusCode)
	}
}

// TestDashboardService_AdminGate: anonymous callers are rejected and nothing
// reaches the owning services.
func TestDashboardService_AdminGate(t *testing.T) {
	authFake := newFakeStatsUpstream(map[string]interface{}{"users": 0})
	defer authFake.close()
	mux := newDashboardMux(dashboardCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	resp, err := http.Get(s.URL + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous GET /dashboard: expected 401, got %d", resp.StatusCode)
	}
	if got := authFake.callCount(); got != 0 {
		t.Errorf("anonymous request reached auth upstream %d times, want 0", got)
	}
}

// TestDashboardService_StatsShape: /dashboard/stats must merge the four
// internal payloads into the exact Phase 26 dashboard JSON contract.
func TestDashboardService_StatsShape(t *testing.T) {
	authFake := newFakeStatsUpstream(map[string]interface{}{
		"users": 5,
		"recentRegistrations": []map[string]interface{}{
			{"date": "2026-09-16", "count": 0}, {"date": "2026-09-17", "count": 1},
		},
	})
	contentFake := newFakeStatsUpstream(map[string]interface{}{"pages": 3, "posts": 4})
	mediaFake := newFakeStatsUpstream(map[string]interface{}{"media": 9})
	messagesFake := newFakeStatsUpstream(map[string]interface{}{
		"messages": 2,
		"recentMessages": []map[string]interface{}{
			{"id": 1, "title": "One", "content": "a", "createdAt": "2026-09-22T10:00:00Z"},
			{"id": 2, "title": "Two", "content": "b", "createdAt": "2026-09-22T11:00:00Z"},
		},
	})
	defer authFake.close()
	defer contentFake.close()
	defer mediaFake.close()
	defer messagesFake.close()

	mux := newDashboardMux(dashboardCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
		ContentURL:    contentFake.server.URL,
		MediaURL:      mediaFake.server.URL,
		MessagesURL:   messagesFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL+"/dashboard/stats", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard/stats: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", resp.StatusCode, readTestBody(resp))
	}
	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)

	if stats["userCount"].(float64) != 5 || stats["pageCount"].(float64) != 3 ||
		stats["blogCount"].(float64) != 4 || stats["mediaCount"].(float64) != 9 {
		t.Errorf("counts wrong: %v", stats)
	}
	regs := stats["recentRegistrations"].([]interface{})
	if len(regs) != 2 || regs[0].(map[string]interface{})["count"].(float64) != 0 {
		t.Errorf("recentRegistrations wrong: %v", regs)
	}
	recent := stats["recentMessages"].([]interface{})
	if len(recent) != 2 || recent[0].(map[string]interface{})["title"] != "One" {
		t.Errorf("recentMessages wrong: %v", recent)
	}

	// All four owning services must have been fetched with the internal token.
	for name, fake := range map[string]*fakeStatsUpstream{
		"auth": authFake, "content": contentFake, "media": mediaFake, "messages": messagesFake,
	} {
		if fake.callCount() != 1 {
			t.Errorf("%s upstream called %d times, want 1", name, fake.callCount())
		}
		if tok := fake.token(); tok != "test-internal-token" {
			t.Errorf("%s upstream token = %q, want test-internal-token", name, tok)
		}
	}
}

// TestDashboardService_Counts: /dashboard returns the plain count contract
// (users/pages/posts/media/messages).
func TestDashboardService_Counts(t *testing.T) {
	authFake := newFakeStatsUpstream(map[string]interface{}{"users": 5})
	contentFake := newFakeStatsUpstream(map[string]interface{}{"pages": 3, "posts": 4})
	mediaFake := newFakeStatsUpstream(map[string]interface{}{"media": 9})
	messagesFake := newFakeStatsUpstream(map[string]interface{}{"messages": 2})
	defer authFake.close()
	defer contentFake.close()
	defer mediaFake.close()
	defer messagesFake.close()

	mux := newDashboardMux(dashboardCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
		ContentURL:    contentFake.server.URL,
		MediaURL:      mediaFake.server.URL,
		MessagesURL:   messagesFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL+"/dashboard", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	defer resp.Body.Close()
	var counts map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&counts)
	if len(counts) != 5 || counts["messages"].(float64) != 2 {
		t.Errorf("counts payload wrong: %v", counts)
	}
	for _, key := range []string{"users", "pages", "posts", "media", "messages"} {
		if _, ok := counts[key]; !ok {
			t.Errorf("missing count key %q", key)
		}
	}
}

// TestDashboardService_UpstreamDown fails loud (502) when an owning service is
// unreachable, so misconfig surfaces instead of a half-empty dashboard.
func TestDashboardService_UpstreamDown(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()

	mux := newDashboardMux(dashboardCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       deadURL,
		ContentURL:    "http://127.0.0.1:1",
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL+"/dashboard", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("unreachable upstream: expected 502, got %d", resp.StatusCode)
	}
}

func readTestBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}
