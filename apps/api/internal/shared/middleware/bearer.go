package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// RequireBearerToken rejects requests without a matching bearer token.
//
// It deliberately performs no network-origin check. Restricting which
// interfaces may reach a handler is the listener's job: behind a reverse proxy
// every RemoteAddr is loopback, so an address check there would report a
// local-only guarantee it cannot enforce. Callers bind a dedicated listener
// instead.
func RequireBearerToken(expected string) func(http.Handler) http.Handler {
	expectedBytes := []byte(expected)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, prefix) {
				writeBearerError(w)
				return
			}
			provided := []byte(strings.TrimSpace(strings.TrimPrefix(header, prefix)))
			if len(provided) != len(expectedBytes) || subtle.ConstantTimeCompare(provided, expectedBytes) != 1 {
				writeBearerError(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeBearerError(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
