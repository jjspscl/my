package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockSessionGetter struct {
	getFn func(ctx context.Context, sessionID string) (string, error)
}

func (m *mockSessionGetter) Get(ctx context.Context, sessionID string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, sessionID)
	}
	return "", errors.New("session not found")
}

func TestRequireAuth_ValidSession_SetsEmailInContext(t *testing.T) {
	sessions := &mockSessionGetter{
		getFn: func(ctx context.Context, sessionID string) (string, error) {
			return "user@test.com", nil
		},
	}

	handler := RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := GetEmailFromContext(r.Context())
		assert.Equal(t, "user@test.com", email)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: "valid-session-id"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuth_MissingCookie_Returns401(t *testing.T) {
	sessions := &mockSessionGetter{}
	handler := RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	assert.Equal(t, "unauthorized", body["error"])
}

func TestRequireAuth_InvalidSession_Returns401(t *testing.T) {
	sessions := &mockSessionGetter{
		getFn: func(ctx context.Context, sessionID string) (string, error) {
			return "", errors.New("session not found")
		},
	}

	handler := RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: "invalid-session"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	assert.Equal(t, "unauthorized", body["error"])
}

func TestRequireAuth_EmptyCookieValue_Returns401(t *testing.T) {
	sessions := &mockSessionGetter{}
	handler := RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: ""})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_SessionError_Returns401(t *testing.T) {
	sessions := &mockSessionGetter{
		getFn: func(ctx context.Context, sessionID string) (string, error) {
			return "", errors.New("redis connection error")
		},
	}

	handler := RequireAuth(sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: "some-session"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetEmailFromContext_NoEmail_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	email := GetEmailFromContext(ctx)
	assert.Empty(t, email)
}

func TestGetEmailFromContext_WithEmail_ReturnsEmail(t *testing.T) {
	ctx := context.WithValue(context.Background(), emailKey, "user@test.com")
	email := GetEmailFromContext(ctx)
	assert.Equal(t, "user@test.com", email)
}

func TestGetEmailFromContext_WrongType_ReturnsEmpty(t *testing.T) {
	ctx := context.WithValue(context.Background(), emailKey, 42)
	email := GetEmailFromContext(ctx)
	assert.Empty(t, email)
}
