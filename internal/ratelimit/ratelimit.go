package ratelimit

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"net/http"
)

// TokenBucket is a concurrency-safe token bucket rate limiter.
type TokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	rate     float64 // tokens per second
	capacity float64
	last     time.Time
}

// NewTokenBucket creates a token bucket with the given rate (tokens/sec) and burst capacity.
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:   float64(burst),
		rate:     rate,
		capacity: float64(burst),
		last:     time.Now(),
	}
}

// Allow returns true if a request may proceed, consuming one token.
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	if b.tokens < 1.0 {
		return false
	}
	b.tokens--
	return true
}

// Limiter wraps a TokenBucket with per-key tracking.
type Limiter struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket
	rate    float64
	burst   int
}

// NewLimiter creates a per-key rate limiter.
func NewLimiter(rate float64, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*TokenBucket),
		rate:    rate,
		burst:   burst,
	}
}

// Allow checks whether the given key may proceed.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	b, ok := l.buckets[key]
	if !ok {
		b = NewTokenBucket(l.rate, l.burst)
		l.buckets[key] = b
	}
	l.mu.Unlock()
	return b.Allow()
}

// ErrRateLimited is returned when a request exceeds the rate limit.
var ErrRateLimited = errors.New("rate limit exceeded")

type rateLimitedResponse struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Middleware is an HTTP middleware that applies per-client rate limiting.
type Middleware struct {
	next  http.Handler
	limit *Limiter
}

// NewMiddleware creates a rate-limiting middleware.
func NewMiddleware(next http.Handler, limit *Limiter) *Middleware {
	return &Middleware{next: next, limit: limit}
}

// clientKey extracts a rate-limit key from the request.
func clientKey(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if parts := strings.Split(fwd, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !m.limit.Allow(clientKey(r)) {
		w.Header().Set("Content-Type", "application/problem+json")
		if rid := r.Header.Get("X-Request-ID"); rid != "" {
			w.Header().Set("X-Request-ID", rid)
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(rateLimitedResponse{
			Type:   "about:blank",
			Title:  http.StatusText(http.StatusTooManyRequests),
			Status: http.StatusTooManyRequests,
			Code:   "RATE_LIMITED",
			Detail: "Too many requests; please slow down",
		})
		return
	}
	m.next.ServeHTTP(w, r)
}

var _ http.Handler = (*Middleware)(nil)
