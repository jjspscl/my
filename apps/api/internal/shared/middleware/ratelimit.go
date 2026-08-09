package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jjspscl/my/internal/shared/response"
)

// slidingWindowLimiter tracks request timestamps per key within a sliding
// window and denies once the count in the window reaches max.
type slidingWindowLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	now     func() time.Time
	entries map[string][]time.Time
}

func newSlidingWindowLimiter(max int, window time.Duration, now func() time.Time) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		max:     max,
		window:  window,
		now:     now,
		entries: make(map[string][]time.Time),
	}
}

// allow records a hit for key and reports whether it is within the limit.
// retryAfter is the duration until the oldest hit in the window expires.
func (l *slidingWindowLimiter) allow(key string) (ok bool, retryAfter time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)
	hits := l.entries[key]

	// Prune hits that have fallen out of the window.
	kept := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.max {
		l.entries[key] = kept
		// With no surviving hits (e.g. max <= 0) there is no oldest hit to
		// anchor Retry-After; report the full window.
		if len(kept) == 0 {
			return false, l.window
		}
		return false, l.window - now.Sub(kept[0])
	}
	l.entries[key] = append(kept, now)
	return true, 0
}

// RateLimit limits requests per client IP within a sliding window. The key is
// r.RemoteAddr; chi's RealIP middleware runs earlier in the chain, so this is
// the real client IP rather than a proxy address. Responds 429 with
// Retry-After when the limit is exceeded.
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimitWithClock(max, window, time.Now)
}

// RateLimitWithClock is RateLimit with an injectable clock for tests.
func RateLimitWithClock(max int, window time.Duration, now func() time.Time) func(http.Handler) http.Handler {
	limiter := newSlidingWindowLimiter(max, window, now)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retryAfter := limiter.allow(r.RemoteAddr)
			if !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
				response.WriteError(w, r, http.StatusTooManyRequests, "rate limit exceeded", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
