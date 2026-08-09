package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit_AllowsUpToMax(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	handler := RateLimitWithClock(3, time.Minute, func() time.Time { return now })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass", i+1)
	}
}

func TestRateLimit_ExceedsMax_Returns429WithRetryAfter(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	handler := RateLimitWithClock(2, time.Minute, func() time.Time { return now })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "60", w.Header().Get("Retry-After"))
	assert.JSONEq(t, `{"error":"rate limit exceeded"}`, w.Body.String())
}

func TestRateLimit_WindowExpiry_AllowsAgain(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	handler := RateLimitWithClock(1, time.Minute, func() time.Time { return now })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Same instant: denied.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// After the window passes: allowed again.
	now = now.Add(2 * time.Minute)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimit_KeysArePerIP(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	handler := RateLimitWithClock(1, time.Minute, func() time.Time { return now })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	reqA.RemoteAddr = "203.0.113.10:1234"
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	reqB.RemoteAddr = "198.51.100.7:4321"

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reqA)
	assert.Equal(t, http.StatusOK, w.Code)

	// Different IP is not affected by A's hit.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, reqB)
	assert.Equal(t, http.StatusOK, w.Code)

	// Same IP as A is now denied.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, reqA)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimit_RetryAfterReflectsOldestHit(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	handler := RateLimitWithClock(2, time.Minute, func() time.Time { return now })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	req.RemoteAddr = "203.0.113.10:1234"

	serve := func() *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	// Hit 1 at t=0, hit 2 at t=30s (both allowed).
	assert.Equal(t, http.StatusOK, serve().Code)
	now = now.Add(30 * time.Second)
	assert.Equal(t, http.StatusOK, serve().Code)

	// Hit 3 at t=45s → denied. Oldest hit (t=0) is still inside the 60s
	// window, so Retry-After is 60 - 45 = 15s.
	now = now.Add(15 * time.Second)
	denied := serve()
	assert.Equal(t, http.StatusTooManyRequests, denied.Code)
	assert.Equal(t, "15", denied.Header().Get("Retry-After"))
}