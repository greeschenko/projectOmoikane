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

// reqRecord captures one internal fan-out request the trash service made.
type reqRecord struct {
	method, path, token string
}

// fakeTrashUpstream is a stand-in for an owning service's /internal/trash*
// surface. It records every request (including the X-Internal-Token header)
// and serves canned items/counts.
type fakeTrashUpstream struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []reqRecord
	items    []map[string]interface{}
	count    int64
}

func newFakeTrashUpstream(items []map[string]interface{}, count int64) *fakeTrashUpstream {
	f := &fakeTrashUpstream{items: items, count: count}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/trash", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(f.items)
			return
		}
		if r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/internal/trash/count", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{"count": f.count})
	})
	mux.HandleFunc("/internal/trash/", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})
	f.server = httptest.NewServer(mux)
	return f
}

func (f *fakeTrashUpstream) record(r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, reqRecord{
		method: r.Method,
		path:   r.URL.Path + optionalQuery(r.URL.RawQuery),
		token:  r.Header.Get("X-Internal-Token"),
	})
}

func optionalQuery(q string) string {
	if q == "" {
		return ""
	}
	return "?" + q
}

func (f *fakeTrashUpstream) close() { f.server.Close() }

func (f *fakeTrashUpstream) calls() []reqRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]reqRecord, len(f.requests))
	copy(out, f.requests)
	return out
}

// adminCookie mints an admin JWT for the trash/dashboard admin gate.
func adminCookie(t *testing.T) *http.Cookie {
	t.Helper()
	token, err := auth.GenerateToken(1, "admin", "test-secret")
	if err != nil {
		t.Fatalf("mint JWT: %v", err)
	}
	return &http.Cookie{Name: "session", Value: token}
}

func TestTrashService_Health(t *testing.T) {
	mux := newTrashMux(trashCfg{JWTSecret: "test-secret"})
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

// TestTrashService_AdminGate: the trash surface must reject anonymous callers
// (the gateway treats /api/trash* as admin-only).
func TestTrashService_AdminGate(t *testing.T) {
	authFake := newFakeTrashUpstream(nil, 0)
	defer authFake.close()
	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	for _, tc := range []struct {
		method, path string
	}{
		{"GET", "/trash"},
		{"GET", "/trash/count"},
		{"POST", "/trash/page/1/restore"},
		{"DELETE", "/trash/page/1"},
	} {
		req, _ := http.NewRequest(tc.method, s.URL+tc.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous: expected 401, got %d", tc.method, tc.path, resp.StatusCode)
		}
	}
	if got := len(authFake.calls()); got != 0 {
		t.Errorf("anonymous request reached upstream %d times, want 0", got)
	}
}

// TestTrashService_Aggregates proves GET /trash fan-out + merge: every
// configured owning service is queried with the shared internal token and the
// items are concatenated.
func TestTrashService_Aggregates(t *testing.T) {
	authFake := newFakeTrashUpstream([]map[string]interface{}{{"id": 1, "title": "User", "entity": "user"}}, 1)
	contentFake := newFakeTrashUpstream([]map[string]interface{}{
		{"id": 2, "title": "Page", "entity": "page"},
		{"id": 3, "title": "Post", "entity": "post"},
	}, 2)
	mediaFake := newFakeTrashUpstream([]map[string]interface{}{{"id": 4, "title": "file.png", "entity": "media"}}, 1)
	messagesFake := newFakeTrashUpstream([]map[string]interface{}{{"id": 5, "title": "Hello", "entity": "contact"}}, 1)
	defer authFake.close()
	defer contentFake.close()
	defer mediaFake.close()
	defer messagesFake.close()

	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
		ContentURL:    contentFake.server.URL,
		MediaURL:      mediaFake.server.URL,
		MessagesURL:   messagesFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL+"/trash", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /trash: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from GET /trash, got %d (%s)", resp.StatusCode, readTestBody(resp))
	}
	var items []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&items)
	if len(items) != 5 {
		t.Errorf("aggregated %d items, want 5", len(items))
	}

	// Every upstream was hit once with the internal token.
	for name, fake := range map[string]*fakeTrashUpstream{
		"auth": authFake, "content": contentFake, "media": mediaFake, "messages": messagesFake,
	} {
		calls := fake.calls()
		if len(calls) != 1 || calls[0].path != "/internal/trash" {
			t.Errorf("%s upstream saw %v, want one GET /internal/trash", name, calls)
		}
		if calls[0].token != "test-internal-token" {
			t.Errorf("%s upstream token = %q, want test-internal-token", name, calls[0].token)
		}
	}
}

func TestTrashService_CountSums(t *testing.T) {
	authFake := newFakeTrashUpstream(nil, 3)
	contentFake := newFakeTrashUpstream(nil, 7)
	mediaFake := newFakeTrashUpstream(nil, 0)
	messagesFake := newFakeTrashUpstream(nil, 2)
	defer authFake.close()
	defer contentFake.close()
	defer mediaFake.close()
	defer messagesFake.close()

	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
		ContentURL:    contentFake.server.URL,
		MediaURL:      mediaFake.server.URL,
		MessagesURL:   messagesFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL+"/trash/count", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /trash/count: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Count int64 `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Count != 12 {
		t.Errorf("summed count = %d, want 12 (3+7+0+2)", out.Count)
	}
}

// TestTrashService_RoutesToOwner proves entity->owner routing: restore and
// hard-delete reach exactly the owning service's internal endpoint.
func TestTrashService_RoutesToOwner(t *testing.T) {
	authFake := newFakeTrashUpstream(nil, 0)
	contentFake := newFakeTrashUpstream(nil, 0)
	mediaFake := newFakeTrashUpstream(nil, 0)
	messagesFake := newFakeTrashUpstream(nil, 0)
	defer authFake.close()
	defer contentFake.close()
	defer mediaFake.close()
	defer messagesFake.close()

	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
		ContentURL:    contentFake.server.URL,
		MediaURL:      mediaFake.server.URL,
		MessagesURL:   messagesFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	act := func(method, path string) (int, string) {
		t.Helper()
		req, _ := http.NewRequest(method, s.URL+path, nil)
		req.AddCookie(adminCookie(t))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		return resp.StatusCode, readTestBody(resp)
	}

	type want struct {
		fake *fakeTrashUpstream
		path string
	}
	cases := []struct {
		method, route string
		want          want
	}{
		{"POST", "/trash/user/11/restore", want{authFake, "/internal/trash/user/11/restore"}},
		{"POST", "/trash/page/12/restore", want{contentFake, "/internal/trash/page/12/restore"}},
		{"DELETE", "/trash/media/13", want{mediaFake, "/internal/trash/media/13"}},
		{"DELETE", "/trash/contact/14", want{messagesFake, "/internal/trash/contact/14"}},
	}
	for _, tc := range cases {
		code, body := act(tc.method, tc.route)
		if code != http.StatusOK {
			t.Errorf("%s %s: expected 200, got %d (%s)", tc.method, tc.route, code, body)
			continue
		}
		calls := tc.want.fake.calls()
		if len(calls) != 1 || calls[0].path != tc.want.path ||
			calls[0].method != tc.method || calls[0].token != "test-internal-token" {
			t.Errorf("%s %s: upstream saw %v, want %s %s with token", tc.method, tc.route, calls, tc.method, tc.want.path)
		}
	}
}

// TestTrashService_EmptyFansOut: DELETE /trash hits every owning service; the
// ?entity= form hits only the owner.
func TestTrashService_EmptyFansOut(t *testing.T) {
	authFake := newFakeTrashUpstream(nil, 0)
	contentFake := newFakeTrashUpstream(nil, 0)
	mediaFake := newFakeTrashUpstream(nil, 0)
	messagesFake := newFakeTrashUpstream(nil, 0)
	defer authFake.close()
	defer contentFake.close()
	defer mediaFake.close()
	defer messagesFake.close()

	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
		ContentURL:    contentFake.server.URL,
		MediaURL:      mediaFake.server.URL,
		MessagesURL:   messagesFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	act := func(method, path string) int {
		t.Helper()
		req, _ := http.NewRequest(method, s.URL+path, nil)
		req.AddCookie(adminCookie(t))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if code := act("DELETE", "/trash"); code != http.StatusOK {
		t.Fatalf("DELETE /trash: expected 200, got %d", code)
	}
	for name, fake := range map[string]*fakeTrashUpstream{
		"auth": authFake, "content": contentFake, "media": mediaFake, "messages": messagesFake,
	} {
		if calls := fake.calls(); len(calls) != 1 || calls[0].method != "DELETE" || calls[0].path != "/internal/trash" {
			t.Errorf("%s upstream saw %v, want one DELETE /internal/trash", name, calls)
		}
	}

	if code := act("DELETE", "/trash?entity=page"); code != http.StatusOK {
		t.Fatalf("DELETE /trash?entity=page: expected 200, got %d", code)
	}
	calls := contentFake.calls()
	last := calls[len(calls)-1]
	if last.method != "DELETE" || last.path != "/internal/trash?entity=page" {
		t.Errorf("entity-limited empty reached content as %v, want DELETE /internal/trash?entity=page", last)
	}
	// The entity-limited empty touches ONLY content: every other service still
	// has exactly one call (from the full empty above).
	for name, fake := range map[string]*fakeTrashUpstream{
		"auth": authFake, "media": mediaFake, "messages": messagesFake,
	} {
		if got := len(fake.calls()); got != 1 {
			t.Errorf("%s upstream saw %d calls after entity-limited empty, want 1", name, got)
		}
	}
}

// TestTrashService_UnknownEntity: a foreign entity type is rejected locally
// (400) and never forwarded.
func TestTrashService_UnknownEntity(t *testing.T) {
	authFake := newFakeTrashUpstream(nil, 0)
	defer authFake.close()
	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       authFake.server.URL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("POST", s.URL+"/trash/widget/1/restore", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /trash/widget/1/restore: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown entity: expected 400, got %d", resp.StatusCode)
	}
	if got := len(authFake.calls()); got != 0 {
		t.Errorf("unknown entity reached upstream %d times, want 0", got)
	}
}

// TestTrashService_UpstreamDown: an unreachable owning service fails loud (502).
func TestTrashService_UpstreamDown(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()

	mux := newTrashMux(trashCfg{
		JWTSecret:     "test-secret",
		InternalToken: "test-internal-token",
		AuthURL:       deadURL,
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL+"/trash", nil)
	req.AddCookie(adminCookie(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /trash: %v", err)
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
