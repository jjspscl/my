package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBearerToken(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		remote string
		status int
	}{
		{name: "missing", remote: "127.0.0.1:1234", status: http.StatusUnauthorized},
		{name: "malformed", header: "Basic abc", remote: "127.0.0.1:1234", status: http.StatusUnauthorized},
		{name: "wrong", header: "Bearer wrong", remote: "127.0.0.1:1234", status: http.StatusUnauthorized},
		{name: "correct", header: "Bearer test-token", remote: "127.0.0.1:1234", status: http.StatusNoContent},
		// A non-loopback RemoteAddr is accepted with a valid token: interface
		// restriction is the listener's responsibility, not this middleware's.
		{name: "remote with valid token", header: "Bearer test-token", remote: "192.0.2.1:1234", status: http.StatusNoContent},
		{name: "remote with wrong token", header: "Bearer wrong", remote: "192.0.2.1:1234", status: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
			req.Header.Set("Authorization", tc.header)
			req.RemoteAddr = tc.remote
			res := httptest.NewRecorder()
			RequireBearerToken("test-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})).ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status = %d, want %d", res.Code, tc.status)
			}
		})
	}
}

func TestRequireBearerTokenSetsChallenge(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	res := httptest.NewRecorder()
	RequireBearerToken("test-token")(http.NotFoundHandler()).ServeHTTP(res, req)

	if got := res.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, "Bearer")
	}
}
