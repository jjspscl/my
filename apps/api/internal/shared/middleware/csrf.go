package middleware

import (
	"net/http"
	"strings"
)

func CSRFProtect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check mutating methods
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			headerToken := r.Header.Get("X-CSRF-Token")
			if headerToken == "" {
				http.Error(w, `{"error":"missing csrf token"}`, http.StatusForbidden)
				return
			}

			cookie, err := r.Cookie("my_csrf")
			if err != nil || !strings.EqualFold(cookie.Value, headerToken) {
				http.Error(w, `{"error":"invalid csrf token"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
