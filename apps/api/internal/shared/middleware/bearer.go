package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

func RequireBearerToken(expected string, localOnly bool) func(http.Handler) http.Handler {
	expectedBytes := []byte(expected)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if localOnly && !isLoopbackRemote(r.RemoteAddr) {
				writeBearerError(w)
				return
			}

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

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeBearerError(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
