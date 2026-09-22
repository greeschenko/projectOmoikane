package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDocsService_Health(t *testing.T) {
	s := httptest.NewServer(newDocsMux())
	defer s.Close()

	resp, err := http.Get(s.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status = %q, want ok", body["status"])
	}
}

// TestDocsService_SwaggerUICtrl: the Swagger UI HTML is served at /swagger/
// (the gateway rewrites the trailing-slash redirect so /api/swagger/ resolves).
func TestDocsService_SwaggerUI(t *testing.T) {
	s := httptest.NewServer(newDocsMux())
	defer s.Close()

	resp, err := http.Get(s.URL + "/swagger/")
	if err != nil {
		t.Fatalf("GET /swagger/: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /swagger/, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatal("swagger UI body empty")
	}
}
