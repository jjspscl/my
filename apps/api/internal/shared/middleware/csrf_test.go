package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFProtect_GET_PassesThrough(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_HEAD_PassesThrough(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodHead, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_OPTIONS_PassesThrough(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_POST_MatchingTokens_Passes(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "test-token-123"})
	req.Header.Set("X-CSRF-Token", "test-token-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_POST_MismatchedTokens_Returns403(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "cookie-token"})
	req.Header.Set("X-CSRF-Token", "different-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	assert.Equal(t, "invalid csrf token", body["error"])
}

func TestCSRFProtect_POST_MissingHeader_Returns403(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "cookie-token"})
	// No X-CSRF-Token header set
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	assert.Equal(t, "missing csrf token", body["error"])
}

func TestCSRFProtect_POST_MissingCookie_Returns403(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.Header.Set("X-CSRF-Token", "some-token")
	// No cookie set
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFProtect_POST_CaseInsensitiveTokenMatch_Passes(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "TokenValue"})
	req.Header.Set("X-CSRF-Token", "tokenvalue") // lowercase — case-insensitive match
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_PUT_RequiresToken(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "token-123"})
	req.Header.Set("X-CSRF-Token", "token-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_DELETE_RequiresToken(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "token-123"})
	req.Header.Set("X-CSRF-Token", "token-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFProtect_PATCH_RequiresToken(t *testing.T) {
	handler := CSRFProtect()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{Name: "my_csrf", Value: "token-123"})
	req.Header.Set("X-CSRF-Token", "token-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
