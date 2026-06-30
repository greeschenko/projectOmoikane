package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_AllowsRequest(t *testing.T) {
	rl := NewRateLimiter(10, 10, time.Minute)
	handler := rl.Middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	resp := httptest.NewRecorder()
	handler(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.Code)
	}
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	rl := NewRateLimiter(1, 3, time.Minute)
	handler := rl.Middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First 3 requests should be allowed (burst = 3)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		resp := httptest.NewRecorder()
		handler(resp, req)
		if resp.Code != http.StatusOK {
			t.Errorf("Request %d: Expected 200, got %d", i+1, resp.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	resp := httptest.NewRecorder()
	handler(resp, req)
	if resp.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", resp.Code)
	}
}

func TestRateLimiter_DifferentIPsNotAffected(t *testing.T) {
	rl := NewRateLimiter(1, 1, time.Minute)
	handler := rl.Middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Exhaust IP1
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	resp1 := httptest.NewRecorder()
	handler(resp1, req1)

	req1b := httptest.NewRequest("GET", "/test", nil)
	req1b.RemoteAddr = "10.0.0.1:12345"
	resp1b := httptest.NewRecorder()
	handler(resp1b, req1b)
	if resp1b.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 should be rate limited, got %d", resp1b.Code)
	}

	// IP2 should still be allowed
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"
	resp2 := httptest.NewRecorder()
	handler(resp2, req2)
	if resp2.Code != http.StatusOK {
		t.Errorf("IP2 should be allowed, got %d", resp2.Code)
	}
}
