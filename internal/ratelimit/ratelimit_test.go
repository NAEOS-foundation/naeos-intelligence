package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokenBucketAllowsUpToBurst(t *testing.T) {
	b := NewTokenBucket(10, 3)
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("token %d should be allowed", i+1)
		}
	}
	if b.Allow() {
		t.Fatal("4th token should be denied (burst exhausted)")
	}
}

func TestTokenBucketRefillsOverTime(t *testing.T) {
	b := NewTokenBucket(10, 1)
	if !b.Allow() {
		t.Fatal("first token should be allowed")
	}
	if b.Allow() {
		t.Fatal("second token should be denied immediately")
	}
	time.Sleep(200 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("token should be refilled after 200ms at 10/s")
	}
}

func TestLimiterPerKey(t *testing.T) {
	l := NewLimiter(100, 1)
	if !l.Allow("client-a") {
		t.Fatal("client-a first request should be allowed")
	}
	if l.Allow("client-a") {
		t.Fatal("client-a second request should be denied (burst=1)")
	}
	if !l.Allow("client-b") {
		t.Fatal("client-b should have independent bucket")
	}
}

func TestMiddlewareRejectsOverLimit(t *testing.T) {
	l := NewLimiter(100, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mw := NewMiddleware(next, l)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request status=%d, want 200", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status=%d, want 429", rec2.Code)
	}
	var body rateLimitedResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "RATE_LIMITED" {
		t.Fatalf("code=%q, want RATE_LIMITED", body.Code)
	}
	if !strings.Contains(rec2.Header().Get("Content-Type"), "application/problem+json") {
		t.Fatalf("content-type=%q, want application/problem+json", rec2.Header().Get("Content-Type"))
	}
}

func TestClientKeyFromXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	req.RemoteAddr = "10.0.0.1:12345"
	if got := clientKey(req); got != "203.0.113.5" {
		t.Fatalf("key=%q, want 203.0.113.5", got)
	}
}

func TestClientKeyFallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.7:44321"
	if got := clientKey(req); got != "198.51.100.7" {
		t.Fatalf("key=%q, want 198.51.100.7", got)
	}
}
